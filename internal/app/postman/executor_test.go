package postman

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeshop/kubetest/pkg/api/executor"
	"github.com/stretchr/testify/assert"
)

func TestPostmanExecutor_StartExecution(t *testing.T) {

	t.Run("runs newman runner command", func(t *testing.T) {
		// given
		executor := GetTestExecutor(t)

		req := httptest.NewRequest(
			"POST",
			"/v1/executions/",
			strings.NewReader(`{"type": "postman/collection", "metadata": {"info":{"name":"KubeTestExampleCollection"}}}`),
		)

		// when
		resp, err := executor.Mux.Test(req)
		assert.NoError(t, err)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
	})

}

func GetTestExecutor(t *testing.T) PostmanExecutor {
	postmanExecutor := NewPostmanExecutor(&RepoMock{
		Object: executor.Execution{Name: "example-execution"},
	})
	postmanExecutor.Init()

	return postmanExecutor
}

// r RepoMock
type RepoMock struct {
	Object executor.Execution
	Error  error
}

func (r *RepoMock) Get(ctx context.Context, id string) (result executor.Execution, err error) {
	return r.Object, r.Error
}

func (r *RepoMock) Insert(ctx context.Context, result executor.Execution) (err error) {
	return r.Error
}

func (r *RepoMock) QueuePull(ctx context.Context) (result executor.Execution, err error) {
	return r.Object, r.Error
}

func (r *RepoMock) Update(ctx context.Context, result executor.Execution) (err error) {
	return r.Error
}
