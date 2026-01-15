package robfig

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

type executorStub struct {
	called bool
	ctxErr error
}

func (e *executorStub) Execute(ctx context.Context, _ *cloud.ScheduleRequest) ([]testkube.TestWorkflowExecution, error) {
	e.called = true
	e.ctxErr = ctx.Err()
	return nil, nil
}

func TestManagerExecuteJobIgnoresCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if ctx.Err() == nil {
		t.Fatal("expected context to be canceled")
	}

	exec := &executorStub{}
	manager := New(zap.NewNop().Sugar(), exec, false)

	job := manager.testWorkflowExecuteJob(ctx, "workflow", "* * * * *", testkube.TestWorkflowCronJobConfig{})
	job()

	if !exec.called {
		t.Fatal("expected executor to be called")
	}
	if exec.ctxErr != nil {
		t.Fatalf("expected uncanceled context, got %v", exec.ctxErr)
	}
}
