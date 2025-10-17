package scheduling

import "context"

type Controller interface {
	StartExecution(ctx context.Context, executionId string) error
	PauseExecution(ctx context.Context, executionId string) error
	ResumeExecution(ctx context.Context, executionId string) error
	AbortExecution(ctx context.Context, executionId string) error
	CancelExecution(ctx context.Context, executionId string) error
	ForceCancelExecution(ctx context.Context, executionId string) error
}
