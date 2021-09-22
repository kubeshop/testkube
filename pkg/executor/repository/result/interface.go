package result

import (
	"context"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

// Repository represent execution result repository
type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (kubtest.Result, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result kubtest.Result) error
	// Update updates execution result
	Update(ctx context.Context, result kubtest.Result) error
	// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
	QueuePull(ctx context.Context) (kubtest.Result, error)
}
