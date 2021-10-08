package result

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Repository represent execution result repository
// TODO try to merge both repositories into one
type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (testkube.Execution, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result testkube.Execution) error
	// Update updates execution result
	Update(ctx context.Context, result testkube.Execution) error
	//UpdateResult updates only result part of execution
	UpdateResult(ctx context.Context, id string, result testkube.ExecutionResult) (err error)
	// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
	QueuePull(ctx context.Context) (testkube.Execution, error)
}
