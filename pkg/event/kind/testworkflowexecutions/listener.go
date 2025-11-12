package testworkflowexecutions

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
)

var _ common.Listener = (*testWorkflowExecutionListener)(nil)

// Update TestWorkflowExecution object in Kubernetes after status change
func NewListener(ctx context.Context, namespace string, kubeClient client.Client) *testWorkflowExecutionListener {
	return &testWorkflowExecutionListener{
		ctx:        ctx,
		namespace:  namespace,
		kubeClient: kubeClient,
	}
}

type testWorkflowExecutionListener struct {
	ctx        context.Context
	namespace  string
	kubeClient client.Client
}

func (l *testWorkflowExecutionListener) Name() string {
	return "TestWorkflowExecution"
}

func (l *testWorkflowExecutionListener) Selector() string {
	return ""
}

func (l *testWorkflowExecutionListener) Kind() string {
	return "TestWorkflowExecution"
}

func (l *testWorkflowExecutionListener) Group() string {
	return "default-group"
}

func (l *testWorkflowExecutionListener) Events() []testkube.EventType {
	return []testkube.EventType{
		testkube.QUEUE_TESTWORKFLOW_EventType,
		testkube.START_TESTWORKFLOW_EventType,
		testkube.END_TESTWORKFLOW_SUCCESS_EventType,
		testkube.END_TESTWORKFLOW_FAILED_EventType,
		testkube.END_TESTWORKFLOW_ABORTED_EventType,
	}
}

func (l *testWorkflowExecutionListener) Metadata() map[string]string {
	return map[string]string{
		"name":     l.Name(),
		"events":   fmt.Sprintf("%v", l.Events()),
		"selector": l.Selector(),
	}
}

func (l *testWorkflowExecutionListener) Match(event testkube.Event) bool {
	_, valid := event.Valid(l.Selector(), l.Events())
	return valid
}

func (l *testWorkflowExecutionListener) Notify(event testkube.Event) testkube.EventResult {
	if event.TestWorkflowExecution == nil || event.TestWorkflowExecution.TestWorkflowExecutionName == "" {
		return testkube.NewSuccessEventResult(event.Id, "ignored")
	}
	l.update(event.TestWorkflowExecution)
	return testkube.NewSuccessEventResult(event.Id, "monitored")
}

func (l *testWorkflowExecutionListener) update(execution *testkube.TestWorkflowExecution) {
	obj := &testworkflowsv1.TestWorkflowExecution{}
	err := l.kubeClient.Get(l.ctx, client.ObjectKey{Name: execution.TestWorkflowExecutionName, Namespace: l.namespace}, obj)
	if err != nil {
		log.DefaultLogger.Errorw("failed to get TestWorkflowExecution resource", "id", execution.Id, "error", err)
		return
	}
	obj.Status = testworkflows.MapTestWorkflowExecutionStatusAPIToKube(execution, obj.Generation)
	err = l.kubeClient.Status().Update(l.ctx, obj)
	if err != nil {
		log.DefaultLogger.Errorw("failed to update TestWorkflowExecution resource", "id", execution.Id, "error", err)
	}
}
