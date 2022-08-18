package v1

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/pkg/api/mock"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/server"
)

func TestParamsNilAssign(t *testing.T) {

	t.Run("merge two maps", func(t *testing.T) {

		p1 := map[string]testkube.Variable{"p1": testkube.NewBasicVariable("p1", "1")}
		p2 := map[string]testkube.Variable{"p2": testkube.NewBasicVariable("p2", "2")}

		out := mergeVariables(p1, p2)

		assert.Equal(t, 2, len(out))
		assert.Equal(t, "1", out["p1"].Value)
	})

	t.Run("merge two maps with override", func(t *testing.T) {

		p1 := map[string]testkube.Variable{"p1": testkube.NewBasicVariable("p1", "1")}
		p2 := map[string]testkube.Variable{"p1": testkube.NewBasicVariable("p1", "2")}

		out := mergeVariables(p1, p2)

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "2", out["p1"].Value)
	})

	t.Run("merge with nil map", func(t *testing.T) {

		p2 := map[string]testkube.Variable{"p2": testkube.NewBasicVariable("p2", "2")}

		out := mergeVariables(nil, p2)

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "2", out["p2"].Value)
	})

}

func TestTestkubeAPI_ExecutionLogsHandler(t *testing.T) {
	app := fiber.New()
	resultRepo := mock.ExecutionResultsRepository{}
	executor := &mock.Executor{}
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		ExecutionResults: &resultRepo,
		Executor:         executor,
	}
	app.Get("/executions/:executionID/logs", s.ExecutionLogsHandler())

	tests := []struct {
		name         string
		route        string
		expectedCode int
		execution    testkube.Execution
		jobLogs      testkube.ExecutorOutput
		wantLogs     string
	}{
		{
			name:         "Test getting execution from result output",
			route:        "/executions/finished-1234/logs",
			expectedCode: 200,
			execution: testkube.Execution{
				Id: "finished-1234",
				ExecutionResult: &testkube.ExecutionResult{
					Status: testkube.StatusPtr(testkube.PASSED_ExecutionStatus),
					Output: "storage logs",
				},
			},
			wantLogs: "storage logs",
		},
		{
			name:         "Test getting execution from job",
			route:        "/executions/running-1234/logs",
			expectedCode: 200,
			execution: testkube.Execution{
				Id: "running-1234",
				ExecutionResult: &testkube.ExecutionResult{
					Status: testkube.StatusPtr(testkube.RUNNING_ExecutionStatus),
				},
			},
			jobLogs: testkube.ExecutorOutput{
				Type_:   output.TypeLogLine,
				Content: "job logs",
			},
			wantLogs: "job logs",
		},
	}
	responsePrefix := "data: "
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultRepo.GetFn = func(ctx context.Context, id string) (testkube.Execution, error) {
				assert.Equal(t, tt.execution.Id, id)

				return tt.execution, nil
			}
			executor.LogsFn = func(id string) (out chan output.Output, err error) {
				assert.Equal(t, tt.execution.Id, id)

				out = make(chan output.Output)
				go func() {
					defer func() {
						close(out)
					}()

					out <- output.Output(tt.jobLogs)
				}()
				return
			}

			req := httptest.NewRequest("GET", tt.route, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode, tt.name)

			b := make([]byte, len(responsePrefix))
			resp.Body.Read(b)
			assert.Equal(t, responsePrefix, string(b))

			var res output.Output
			err = json.NewDecoder(resp.Body).Decode(&res)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantLogs, res.Content)
		})
	}
}
