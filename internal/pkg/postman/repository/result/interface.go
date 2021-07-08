package result

import (
	"context"

	"github.com/kubeshop/kubetest/pkg/api/executor"
)

type Repository interface {
	Get(ctx context.Context, id string) (executor.ExecutionResult, error)
	Insert(ctx context.Context, result executor.ExecutionResult) error
}
