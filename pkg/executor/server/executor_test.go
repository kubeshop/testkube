package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestCypressExecutor_StartExecution(t *testing.T) {

	t.Run("runs newman runner command", func(t *testing.T) {
		// given
		executor := GetTestExecutor(t)

		req := httptest.NewRequest(
			"POST",
			"/v1/executions/",
			strings.NewReader(`{"type": "cypress/collection", "metadata": "{\"info\":{\"name\":\"testkubeExampleCollection\"}}"}`),
		)

		// when
		resp, err := executor.Mux.Test(req)
		assert.NoError(t, err)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
	})

}

func GetTestExecutor(t *testing.T) Executor {
	cypressExecutor := NewExecutor(
		&RepoMock{
			Object: testkube.Execution{Id: "1"},
		},
		&ExampleRunner{},
	)
	cypressExecutor.Init()

	return cypressExecutor
}

// r RepoMock
type RepoMock struct {
	Object testkube.Execution
	Error  error
}

func (r *RepoMock) Get(ctx context.Context, id string) (result testkube.Execution, err error) {
	return r.Object, r.Error
}

func (r *RepoMock) Insert(ctx context.Context, result testkube.Execution) (err error) {
	return r.Error
}

func (r *RepoMock) QueuePull(ctx context.Context) (result testkube.Execution, err error) {
	return r.Object, r.Error
}

func (r *RepoMock) UpdateResult(ctx context.Context, id string, result testkube.ExecutionResult) (err error) {
	return r.Error
}

func (r *RepoMock) Update(ctx context.Context, result testkube.Execution) (err error) {
	return r.Error
}

// ExampleRunner for template - change me to some valid runner
type ExampleRunner struct {
}

func (r *ExampleRunner) Run(execution testkube.Execution) (testkube.ExecutionResult, error) {
	return testkube.ExecutionResult{
		Status: testkube.ExecutionStatusSuccess,
		Output: "exmaple test output",
	}, nil
}
