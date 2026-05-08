package robfig

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cronjob"
)

type recordingExecutor struct {
	calls int
	ctx   context.Context
	req   *cloud.ScheduleRequest
}

func (e *recordingExecutor) Execute(ctx context.Context, req *cloud.ScheduleRequest) ([]testkube.TestWorkflowExecution, error) {
	e.calls++
	e.ctx = ctx
	e.req = req
	return nil, nil
}

func TestManagerStartStopResetsExecContext(t *testing.T) {
	logger := zap.NewNop().Sugar()
	executor := &recordingExecutor{}
	manager := New(logger, executor, false)

	oldCtx := manager.execCtx
	manager.Start()

	select {
	case <-oldCtx.Done():
	default:
		t.Fatal("expected previous exec context to be canceled on start")
	}

	if err := manager.execCtx.Err(); err != nil {
		t.Fatalf("expected new exec context to be active, got %v", err)
	}

	currentCtx := manager.execCtx
	manager.Stop()

	select {
	case <-currentCtx.Done():
	default:
		t.Fatal("expected exec context to be canceled on stop")
	}
}

func TestJobUsesManagerContext(t *testing.T) {
	logger := zap.NewNop().Sugar()
	executor := &recordingExecutor{}
	manager := New(logger, executor, false)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	workflow := cronjob.Workflow{Name: "workflow-a", EnvId: "env-1"}
	config := testkube.TestWorkflowCronJobConfig{Cron: "* * * * *"}

	if err := manager.ReplaceWorkflowSchedules(canceledCtx, workflow, []testkube.TestWorkflowCronJobConfig{config}); err != nil {
		t.Fatalf("unexpected error setting schedules: %v", err)
	}

	entryID := manager.cronEntries[workflow.Name][cronSpec(config)]
	entry := manager.cron.Entry(entryID)
	if entry.Job == nil {
		t.Fatal("expected scheduled job to be registered")
	}

	entry.Job.Run()

	if executor.calls != 1 {
		t.Fatalf("expected executor to be called once, got %d", executor.calls)
	}
	if executor.ctx == nil {
		t.Fatal("expected executor context to be set")
	}
	if err := executor.ctx.Err(); err != nil {
		t.Fatalf("expected executor context to be active, got %v", err)
	}
	if executor.ctx == canceledCtx {
		t.Fatal("expected executor to use manager context, got canceled caller context")
	}
}
