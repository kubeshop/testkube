package postman

import (
	"context"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeshop/kubetest/pkg/api/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type RunnerMock struct {
	Error  error
	Result string
	T      *testing.T
}

func (r RunnerMock) Run(input io.Reader) (string, error) {
	body, err := ioutil.ReadAll(input)
	require.NoError(r.T, err)
	require.Contains(r.T, string(body), "KubeTestExampleCollection")
	return r.Result, r.Error
}

func GetTestExecutor(t *testing.T) PostmanExecutor {
	postmanExecutor := NewPostmanExecutor()
	postmanExecutor.Runner = &RunnerMock{
		Result: "TEST COMPLETED",
		T:      t,
	}
	postmanExecutor.Repository = &RepoMock{
		Object: executor.Execution{Name: "example-execution"},
	}

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
