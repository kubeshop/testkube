// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowexecutor

import (
	"bufio"
	"context"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
)

//go:generate mockgen -destination=./mock_executor.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowexecutor" TestWorkflowExecutor
type TestWorkflowExecutor interface {
	Schedule(bundle *testworkflowprocessor.Bundle, execution testkube.TestWorkflowExecution)
	Control(ctx context.Context, execution testkube.TestWorkflowExecution)
	Recover(ctx context.Context)
}

type executor struct {
	emitter    *event.Emitter
	clientSet  kubernetes.Interface
	repository testworkflow.Repository
	output     testworkflow.OutputRepository
	namespace  string
}

func New(emitter *event.Emitter, clientSet kubernetes.Interface, repository testworkflow.Repository, output testworkflow.OutputRepository, namespace string) TestWorkflowExecutor {
	return &executor{
		emitter:    emitter,
		clientSet:  clientSet,
		repository: repository,
		output:     output,
		namespace:  namespace,
	}
}

func (e *executor) Schedule(bundle *testworkflowprocessor.Bundle, execution testkube.TestWorkflowExecution) {
	// Inform about execution start
	e.emitter.Notify(testkube.NewEventQueueTestWorkflow(&execution))

	// Deploy required resources
	err := e.Deploy(context.Background(), bundle)
	if err != nil {
		e.handleFatalError(execution, err, time.Time{})
		return
	}

	// Start to control the results
	go e.Control(context.Background(), execution)
}

func (e *executor) Deploy(ctx context.Context, bundle *testworkflowprocessor.Bundle) (err error) {
	for _, item := range bundle.Secrets {
		_, err = e.clientSet.CoreV1().Secrets(e.namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return
		}
	}
	for _, item := range bundle.ConfigMaps {
		_, err = e.clientSet.CoreV1().ConfigMaps(e.namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return
		}
	}
	_, err = e.clientSet.BatchV1().Jobs(e.namespace).Create(ctx, &bundle.Job, metav1.CreateOptions{})
	return
}

func (e *executor) handleFatalError(execution testkube.TestWorkflowExecution, err error, ts time.Time) {
	// Detect error type
	isAborted := errors.Is(err, testworkflowcontroller.ErrJobAborted)
	isTimeout := errors.Is(err, testworkflowcontroller.ErrJobTimeout)

	// Build error timestamp, adjusting it for aborting job
	if ts.IsZero() {
		ts = time.Now()
		if isAborted || isTimeout {
			ts = ts.Truncate(testworkflowcontroller.JobRetrievalTimeout)
		}
	}

	// Apply the expected result
	execution.Result.Fatal(err, isAborted, ts)
	err = e.repository.UpdateResult(context.Background(), execution.Id, execution.Result)
	if err != nil {
		log.DefaultLogger.Errorf("failed to save fatal error for execution %s: %v", execution.Id, err)
	}
	e.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&execution))
	go testworkflowcontroller.Cleanup(context.Background(), e.clientSet, e.namespace, execution.Id)
}

func (e *executor) Recover(ctx context.Context) {
	list, err := e.repository.GetRunning(ctx)
	if err != nil {
		return
	}
	for _, execution := range list {
		e.Control(context.Background(), execution)
	}
}

func (e *executor) Control(ctx context.Context, execution testkube.TestWorkflowExecution) {
	ctrl, err := testworkflowcontroller.New(ctx, e.clientSet, e.namespace, execution.Id, execution.ScheduledAt)
	if err != nil {
		e.handleFatalError(execution, err, time.Time{})
		return
	}

	// Prepare stream for writing log
	r, writer := io.Pipe()
	reader := bufio.NewReader(r)
	ref := ""

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for v := range ctrl.Watch(ctx).Stream(ctx).Channel() {
			if v.Error != nil {
				continue
			}
			if v.Value.Output != nil {
				execution.Output = append(execution.Output, *v.Value.Output.ToInternal())
			} else if v.Value.Result != nil {
				execution.Result = v.Value.Result
				if execution.Result.IsFinished() {
					execution.StatusAt = execution.Result.FinishedAt
				}
				err := e.repository.UpdateResult(ctx, execution.Id, execution.Result)
				if err != nil {
					log.DefaultLogger.Error(errors.Wrap(err, "error saving test workflow execution result"))
				}
			} else {
				if ref != v.Value.Ref {
					ref = v.Value.Ref
					_, err := writer.Write([]byte(data.SprintHint(ref, "start")))
					if err != nil {
						log.DefaultLogger.Error(errors.Wrap(err, "saving log output signature"))
					}
				}
				_, err := writer.Write([]byte(v.Value.Log))
				if err != nil {
					log.DefaultLogger.Error(errors.Wrap(err, "saving log output content"))
				}
			}
		}

		// Try to gracefully handle abort
		if execution.Result.FinishedAt.IsZero() {
			// Handle container failure
			abortedAt := time.Time{}
			for _, v := range execution.Result.Steps {
				if v.Status != nil && *v.Status == testkube.ABORTED_TestWorkflowStepStatus {
					abortedAt = v.FinishedAt
					break
				}
			}
			if !abortedAt.IsZero() {
				e.handleFatalError(execution, testworkflowcontroller.ErrJobAborted, abortedAt)
			} else {
				// Handle unknown state
				ctrl, err = testworkflowcontroller.New(ctx, e.clientSet, e.namespace, execution.Id, execution.ScheduledAt)
				if err == nil {
					for v := range ctrl.Watch(ctx).Stream(ctx).Channel() {
						if v.Error != nil || v.Value.Output == nil {
							continue
						}

						execution.Result = v.Value.Result
						if execution.Result.IsFinished() {
							execution.StatusAt = execution.Result.FinishedAt
						}
						err := e.repository.UpdateResult(ctx, execution.Id, execution.Result)
						if err != nil {
							log.DefaultLogger.Error(errors.Wrap(err, "error saving test workflow execution result"))
						}
					}
				} else {
					e.handleFatalError(execution, err, time.Time{})
				}
			}
		}

		err := writer.Close()
		if err != nil {
			log.DefaultLogger.Errorw("failed to close TestWorkflow log output stream", "id", execution.Id, "error", err)
		}

		// TODO: Consider AppendOutput ($push) instead
		_ = e.repository.UpdateOutput(ctx, execution.Id, execution.Output)
		if execution.Result.IsFinished() {
			if execution.Result.IsPassed() {
				e.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(&execution))
			} else if execution.Result.IsAborted() {
				e.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(&execution))
			} else {
				e.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&execution))
			}
		}
	}()

	// Stream the log into Minio
	err = e.output.SaveLog(context.Background(), execution.Id, execution.Workflow.Name, reader)
	if err != nil {
		log.DefaultLogger.Errorw("failed to save TestWorkflow log output", "id", execution.Id, "error", err)
	}

	wg.Wait()

	err = testworkflowcontroller.Cleanup(ctx, e.clientSet, e.namespace, execution.Id)
	if err != nil {
		log.DefaultLogger.Errorw("failed to cleanup TestWorkflow resources", "id", execution.Id, "error", err)
	}
}
