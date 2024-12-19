package testworkflowexecutionmetrics

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

var _ common.Listener = (*testWorkflowExecutionMetricsListener)(nil)

// Send metrics based on the Test Workflow Execution status changes
func NewListener(ctx context.Context, metrics metrics.Metrics, dashboardURI string) *testWorkflowExecutionMetricsListener {
	return &testWorkflowExecutionMetricsListener{
		ctx:          ctx,
		metrics:      metrics,
		dashboardURI: dashboardURI,
	}
}

type testWorkflowExecutionMetricsListener struct {
	ctx          context.Context
	metrics      metrics.Metrics
	dashboardURI string
}

func (l *testWorkflowExecutionMetricsListener) Name() string {
	return "TestWorkflowExecutionMetrics"
}

func (l *testWorkflowExecutionMetricsListener) Selector() string {
	return ""
}

func (l *testWorkflowExecutionMetricsListener) Kind() string {
	return "TestWorkflowExecutionMetrics"
}

func (l *testWorkflowExecutionMetricsListener) Events() []testkube.EventType {
	return []testkube.EventType{
		testkube.END_TESTWORKFLOW_SUCCESS_EventType,
		testkube.END_TESTWORKFLOW_FAILED_EventType,
		testkube.END_TESTWORKFLOW_ABORTED_EventType,
	}
}

func (l *testWorkflowExecutionMetricsListener) Metadata() map[string]string {
	return map[string]string{
		"name":     l.Name(),
		"events":   fmt.Sprintf("%v", l.Events()),
		"selector": l.Selector(),
	}
}

func (l *testWorkflowExecutionMetricsListener) Notify(event testkube.Event) testkube.EventResult {
	if event.TestWorkflowExecution == nil {
		return testkube.NewSuccessEventResult(event.Id, "ignored")
	}
	l.metrics.IncAndObserveExecuteTestWorkflow(*event.TestWorkflowExecution, l.dashboardURI)
	return testkube.NewSuccessEventResult(event.Id, "monitored")
}
