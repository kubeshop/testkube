package sequence

import (
	"context"
)

type ExecutionType string

const (
	ExecutionTypeTest         ExecutionType = "t"
	ExecutionTypeTestSuite    ExecutionType = "ts"
	ExecutionTypeTestWorkflow ExecutionType = "tw"
)

//go:generate mockgen -destination=./mock_repository.go -package=sequence "github.com/kubeshop/testkube/pkg/repository/sequence" Repository
type Repository interface {
	// GetNextExecutionNumber gets next execution number by name and type
	GetNextExecutionNumber(ctx context.Context, name string, executionType ExecutionType) (number int32, err error)
	// DeleteExecutionNumber deletes execution number by name and type
	DeleteExecutionNumber(ctx context.Context, name string, executionType ExecutionType) (err error)
	// DeleteExecutionNumbers deletes multiple execution numbers by names and type
	DeleteExecutionNumbers(ctx context.Context, names []string, executionType ExecutionType) (err error)
	// DeleteAllExecutionNumbers deletes all execution numbers by type
	DeleteAllExecutionNumbers(ctx context.Context, executionType ExecutionType) (err error)
}
