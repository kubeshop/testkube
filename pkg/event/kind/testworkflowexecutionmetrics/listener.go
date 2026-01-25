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

func (l *testWorkflowExecutionMetricsListener) Group() string {
	return ""
}

func (l *testWorkflowExecutionMetricsListener) Events() []testkube.EventType {
	return []testkube.EventType{
		testkube.END_TESTWORKFLOW_SUCCESS_EventType,
		testkube.END_TESTWORKFLOW_FAILED_EventType,
		testkube.END_TESTWORKFLOW_ABORTED_EventType,
		testkube.END_TESTWORKFLOW_CANCELED_EventType,
	}
}

func (l *testWorkflowExecutionMetricsListener) Metadata() map[string]string {
	return map[string]string{
		"name":     l.Name(),
		"events":   fmt.Sprintf("%v", l.Events()),
		"selector": l.Selector(),
	}
}

func (l *testWorkflowExecutionMetricsListener) Match(event testkube.Event) bool {
	_, valid := event.Valid(l.Group(), l.Selector(), l.Events())
	return valid
}

func (l *testWorkflowExecutionMetricsListener) Notify(event testkube.Event) testkube.EventResult {
	if event.TestWorkflowExecution == nil {
		return testkube.NewSuccessEventResult(event.Id, "ignored")
	}

	// Check if metrics are silenced for this execution
	if event.TestWorkflowExecution.SilentMode != nil && event.TestWorkflowExecution.SilentMode.Metrics {
		return testkube.NewSuccessEventResult(event.Id, "metrics silenced for test workflow execution")
	}

	l.metrics.IncAndObserveExecuteTestWorkflow(*event.TestWorkflowExecution, l.dashboardURI)
	return testkube.NewSuccessEventResult(event.Id, "monitored")
}
