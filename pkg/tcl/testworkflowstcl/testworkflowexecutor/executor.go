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
		e.handleFatalError(execution, err)
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

func (e *executor) handleFatalError(execution testkube.TestWorkflowExecution, err error) {
	execution.Result.Fatal(err)
	err = e.repository.UpdateResult(context.Background(), execution.Id, execution.Result)
	e.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&execution))
	go testworkflowcontroller.Cleanup(context.Background(), e.clientSet, e.namespace, execution.Id)
}

func (e *executor) Control(ctx context.Context, execution testkube.TestWorkflowExecution) {
	ctrl, err := testworkflowcontroller.New(ctx, e.clientSet, e.namespace, execution.Id, execution.ScheduledAt)
	if err != nil {
		e.handleFatalError(execution, err)
		return
	}

	// Prepare stream for writing log
	r, writer := io.Pipe()
	reader := bufio.NewReader(r)
	ref := ""

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
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
				_ = e.repository.UpdateResult(ctx, execution.Id, execution.Result)
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

		err := writer.Close()
		if err != nil {
			log.DefaultLogger.Error(errors.Wrap(err, "saving log output - closing stream"))
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
		log.DefaultLogger.Error(errors.Wrap(err, "saving log output"))
	}

	wg.Wait()
	testworkflowcontroller.Cleanup(ctx, e.clientSet, e.namespace, execution.Id)
}
