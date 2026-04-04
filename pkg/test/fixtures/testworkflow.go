package fixtures

import (
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func Ptr[T any](v T) *T {
	return &v
}

type ExecutionOption func(*testkube.TestWorkflowExecution)

func WithStatus(status testkube.TestWorkflowStatus) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		if e.Result == nil {
			e.Result = &testkube.TestWorkflowResult{
				Status:          Ptr(status),
				PredictedStatus: Ptr(status),
			}
		} else {
			e.Result.Status = Ptr(status)
			e.Result.PredictedStatus = Ptr(status)
		}
		switch status {
		case testkube.PASSED_TestWorkflowStatus, testkube.FAILED_TestWorkflowStatus,
			testkube.ABORTED_TestWorkflowStatus, testkube.CANCELED_TestWorkflowStatus:
			if e.Result.FinishedAt.IsZero() {
				e.Result.FinishedAt = time.Now()
				e.StatusAt = e.Result.FinishedAt
			}
		case testkube.RUNNING_TestWorkflowStatus:
			if e.Result.StartedAt.IsZero() {
				e.Result.StartedAt = time.Now()
			}
		}
	}
}

func WithTags(tags map[string]string) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.Tags = tags
	}
}

func WithLabels(labels map[string]string) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		if e.Workflow != nil {
			e.Workflow.Labels = labels
		}
	}
}

func WithRunnerID(id string) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.RunnerId = id
	}
}

func WithGroupID(id string) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.GroupId = id
	}
}

func WithNumber(n int32) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.Number = n
	}
}

func WithScheduledAt(t time.Time) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.ScheduledAt = t
	}
}

func WithResult(result *testkube.TestWorkflowResult) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.Result = result
	}
}

func WithRunningContext(rc *testkube.TestWorkflowRunningContext) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.RunningContext = rc
	}
}

func WithConfigParams(cp map[string]testkube.TestWorkflowExecutionConfigValue) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		e.ConfigParams = cp
	}
}

func WithInitialization(init *testkube.TestWorkflowStepResult) ExecutionOption {
	return func(e *testkube.TestWorkflowExecution) {
		if e.Result != nil {
			e.Result.Initialization = init
		}
	}
}

func NewExecution(name string, opts ...ExecutionOption) testkube.TestWorkflowExecution {
	e := testkube.TestWorkflowExecution{
		Id:   fmt.Sprintf("exec-%s-%d", name, time.Now().UnixNano()),
		Name: name,
		Workflow: &testkube.TestWorkflow{
			Name: name,
			Spec: &testkube.TestWorkflowSpec{},
		},
		ScheduledAt: time.Now(),
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func NewQueuedExecution(name string, opts ...ExecutionOption) testkube.TestWorkflowExecution {
	return NewExecution(name, append([]ExecutionOption{WithStatus(testkube.QUEUED_TestWorkflowStatus)}, opts...)...)
}

func NewRunningExecution(name string, opts ...ExecutionOption) testkube.TestWorkflowExecution {
	return NewExecution(name, append([]ExecutionOption{WithStatus(testkube.RUNNING_TestWorkflowStatus)}, opts...)...)
}

func NewFinishedExecution(name string, status testkube.TestWorkflowStatus, opts ...ExecutionOption) testkube.TestWorkflowExecution {
	return NewExecution(name, append([]ExecutionOption{WithStatus(status)}, opts...)...)
}

func NewPassedExecution(name string, opts ...ExecutionOption) testkube.TestWorkflowExecution {
	return NewFinishedExecution(name, testkube.PASSED_TestWorkflowStatus, opts...)
}

func NewFailedExecution(name string, opts ...ExecutionOption) testkube.TestWorkflowExecution {
	return NewFinishedExecution(name, testkube.FAILED_TestWorkflowStatus, opts...)
}

func ResultWithDuration(durationMs int32) *testkube.TestWorkflowResult {
	return &testkube.TestWorkflowResult{
		Status:          Ptr(testkube.PASSED_TestWorkflowStatus),
		PredictedStatus: Ptr(testkube.PASSED_TestWorkflowStatus),
		DurationMs:      durationMs,
		TotalDurationMs: durationMs,
		FinishedAt:      time.Now(),
	}
}
