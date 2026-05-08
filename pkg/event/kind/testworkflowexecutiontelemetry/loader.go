package testworkflowexecutiontelemetry

import (
	"context"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/repository/config"
)

var _ common.ListenerLoader = (*testWorkflowExecutionTelemetryLoader)(nil)

func NewLoader(ctx context.Context, configRepository config.Repository) *testWorkflowExecutionTelemetryLoader {
	return &testWorkflowExecutionTelemetryLoader{
		listener: NewListener(ctx, configRepository),
	}
}

type testWorkflowExecutionTelemetryLoader struct {
	listener *testWorkflowExecutionTelemetryListener
}

func (r *testWorkflowExecutionTelemetryLoader) Kind() string {
	return "TestWorkflowExecutionTelemetry"
}

func (r *testWorkflowExecutionTelemetryLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{r.listener}, nil
}
