package postman

import (
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeshop/kubetest/pkg/runner"
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
			strings.NewReader(`{"type": "postman-collection", "metadata": {"info":{"name":"KubeTestExampleCollection"}}}`),
		)

		// when
		resp, err := executor.Mux.Test(req)
		assert.NoError(t, err)

		// then
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode, 200)
	})

}

type RunnerMock struct {
	Error  error
	Result runner.Result
	T      *testing.T
}

func (r RunnerMock) Run(input io.Reader) (runner.Result, error) {
	body, err := ioutil.ReadAll(input)
	require.NoError(r.T, err)
	require.Contains(r.T, string(body), "KubeTestExampleCollection")
	return r.Result, r.Error
}

func GetTestExecutor(t *testing.T) PostmanExecutor {
	executor := NewPostmanExecutor()
	executor.Runner = &RunnerMock{
		Result: runner.Result{
			Output: "TEST COMPLETED",
		},
		T: t,
	}

	executor.Init()

	return executor
}
