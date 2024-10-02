package executor

import "context"

type Command string

//go:generate mockgen -destination=./mock_executor.go -package=executor "github.com/kubeshop/testkube/pkg/cloud/data/executor" Executor
type Executor interface {
	Execute(ctx context.Context, command Command, payload any) (response []byte, err error)
	Close() error
}
