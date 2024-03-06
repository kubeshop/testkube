// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowexecutor

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
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
	namespace  string
}

func New(emitter *event.Emitter, clientSet kubernetes.Interface, repository testworkflow.Repository, namespace string) TestWorkflowExecutor {
	return &executor{
		emitter:    emitter,
		clientSet:  clientSet,
		repository: repository,
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

	for v := range ctrl.Watch(ctx).Stream(ctx).Channel() {
		if v.Error != nil {
			continue
		}
		if v.Value.Output != nil {
			execution.Output = append(execution.Output, *v.Value.Output.ToInternal())
		}
		if v.Value.Result != nil {
			execution.Result = v.Value.Result
			if execution.Result.IsFinished() {
				execution.StatusAt = execution.Result.FinishedAt
			}
			_ = e.repository.UpdateResult(ctx, execution.Id, execution.Result)
		}
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
	testworkflowcontroller.Cleanup(ctx, e.clientSet, e.namespace, execution.Id)
}
