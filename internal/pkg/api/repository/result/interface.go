package result

import (
	"context"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (kubtest.ScriptExecution, error)
	// GetByName gets execution result by name
	GetByNameAndScript(ctx context.Context, name, script string) (kubtest.ScriptExecution, error)
	// GetNewestExecutions gets top X newest executions
	GetNewestExecutions(ctx context.Context, limit int) ([]kubtest.ScriptExecution, error)
	// GetScriptExecutions gets executions for given script ID
	GetScriptExecutions(ctx context.Context, scriptID string) ([]kubtest.ScriptExecution, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result kubtest.ScriptExecution) error
	// Update updates execution result
	Update(ctx context.Context, result kubtest.ScriptExecution) error
}
