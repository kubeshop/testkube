package testworkflowexecutionmetrics

import (
	"context"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

var _ common.ListenerLoader = (*testWorkflowExecutionMetricsLoader)(nil)

func NewLoader(ctx context.Context, metrics metrics.Metrics, dashboardURI string) *testWorkflowExecutionMetricsLoader {
	return &testWorkflowExecutionMetricsLoader{
		listener: NewListener(ctx, metrics, dashboardURI),
	}
}

type testWorkflowExecutionMetricsLoader struct {
	listener *testWorkflowExecutionMetricsListener
}

func (r *testWorkflowExecutionMetricsLoader) Kind() string {
	return "TestWorkflowExecutionMetrics"
}

func (r *testWorkflowExecutionMetricsLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{r.listener}, nil
}
