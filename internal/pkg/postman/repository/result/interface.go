package result

import (
	"context"

	"github.com/kubeshop/kubetest/pkg/api/executor"
)

type Repository interface {
	Get(ctx context.Context, id string) (executor.Execution, error)
	Insert(ctx context.Context, result executor.Execution) error
}
