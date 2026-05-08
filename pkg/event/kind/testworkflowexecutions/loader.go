package testworkflowexecutions

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

var _ common.ListenerLoader = (*testWorkflowExecutionLoader)(nil)

func NewLoader(ctx context.Context, namespace string, kubeClient client.Client) *testWorkflowExecutionLoader {
	return &testWorkflowExecutionLoader{
		listener: NewListener(ctx, namespace, kubeClient),
	}
}

type testWorkflowExecutionLoader struct {
	listener *testWorkflowExecutionListener
}

func (r *testWorkflowExecutionLoader) Kind() string {
	return "TestWorkflowExecution"
}

func (r *testWorkflowExecutionLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{r.listener}, nil
}
