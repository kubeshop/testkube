package result

import (
	"context"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (kubtest.Execution, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result kubtest.Execution) error
	// Update updates execution result
	Update(ctx context.Context, result kubtest.Execution) error
	// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
	QueuePull(ctx context.Context) (kubtest.Execution, error)
}
