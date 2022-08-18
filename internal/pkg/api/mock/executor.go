package mock

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

type Executor struct {
	LogsFn func(id string) (chan output.Output, error)
}

func (e Executor) Watch(id string) chan client.ResultEvent {
	panic("not implemented")
}

func (e Executor) Get(id string) (testkube.ExecutionResult, error) {
	panic("not implemented")
}

func (e Executor) Execute(execution testkube.Execution, options client.ExecuteOptions) (testkube.ExecutionResult, error) {
	panic("not implemented")
}

func (e Executor) ExecuteSync(execution testkube.Execution, options client.ExecuteOptions) (testkube.ExecutionResult, error) {
	panic("not implemented")
}

func (e Executor) Abort(id string) *testkube.ExecutionResult {
	panic("not implemented")
}

func (e Executor) Logs(id string) (chan output.Output, error) {
	if e.LogsFn == nil {
		panic("not implemented")
	}
	return e.LogsFn(id)
}
