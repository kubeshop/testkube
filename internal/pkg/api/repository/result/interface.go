package result

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (testkube.Execution, error)
	// GetByName gets execution result by name
	GetByNameAndScript(ctx context.Context, name, script string) (testkube.Execution, error)
	// GetNewestExecutions gets top X newest executions
	GetNewestExecutions(ctx context.Context, limit int) ([]testkube.Execution, error)
	// GetExecutions gets executions for given script ID
	GetExecutions(ctx context.Context, scriptID string) ([]testkube.Execution, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result testkube.Execution) error
	// Update updates execution result
	Update(ctx context.Context, result testkube.Execution) error
	// UpdateExecution updates result in execution
	UpdateResult(ctx context.Context, id string, execution testkube.ExecutionResult) error
	// StartExecution updates execution start time
	StartExecution(ctx context.Context, id string, startTime time.Time) error
	// EndExecution updates execution end time
	EndExecution(ctx context.Context, id string, endTime time.Time) error
}
