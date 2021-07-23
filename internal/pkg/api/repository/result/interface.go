package result

import (
	"context"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (kubetest.Execution, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result kubetest.Execution) error
	// Update updates execution result
	Update(ctx context.Context, result kubetest.Execution) error
}
