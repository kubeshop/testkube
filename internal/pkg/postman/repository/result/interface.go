package result

import (
	"context"

	"github.com/kubeshop/kubetest/pkg/api/executor"
)

type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (executor.Execution, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result executor.Execution) error
	// Update updates execution result
	Update(ctx context.Context, result executor.Execution) error
	// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
	QueuePull(ctx context.Context) (executor.Execution, error)
}
