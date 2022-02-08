package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestScriptsAPI(t *testing.T) {

	t.Run("Execute script with given ID", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"1", "executionResult":{"status": "success", "output":"execution completed"}}`)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		execution, err := client.ExecuteScript("test", "testkube", "some name", map[string]string{}, "")

		// then
		assert.Equal(t, "1", execution.Id)
		assert.Equal(t, testkube.SUCCESS_ExecutionStatus, *execution.ExecutionResult.Status)
		assert.Equal(t, "execution completed", execution.ExecutionResult.Output)
		assert.NoError(t, err)
	})

	t.Run("Get executed script details", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/executions/1", r.URL.Path)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"1", "executionResult":{"status": "error"}}`)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		execution, err := client.GetExecution("1")

		// then
		assert.Equal(t, "1", execution.Id)
		assert.Equal(t, testkube.ERROR__ExecutionStatus, *execution.ExecutionResult.Status)
		assert.NoError(t, err)
	})

	t.Run("List scripts executions", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintf(w, `{"totals":{"results":2, "passed":1, "failed":1},"results":[{"id":"1", "executionResult":{"status": "error"}}, {"id":"2", "executionResult":{"status":"error"}}]}`)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		response, err := client.ListExecutions("test", 0, nil)

		// then
		assert.Equal(t, int32(2), response.Totals.Results)
		assert.Equal(t, int32(1), response.Totals.Failed)
		assert.Equal(t, int32(1), response.Totals.Passed)
		assert.Len(t, response.Results, 2)

		assert.Equal(t, "1", response.Results[0].Id)
		assert.Equal(t, "2", response.Results[1].Id)
		assert.NoError(t, err)
	})

	t.Run("Create script", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"1", "name":"t1", "content":{"data":"{}"}, "type":"postman/collection"}`)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		response, err := client.CreateScript(UpsertScriptOptions{
			Content: testkube.NewStringScriptContent("{}"),
			Name:    "t1",
			Type_:   "postman/collection",
		})

		// then
		assert.NoError(t, err)
		assert.Equal(t, "{}", response.Content.Data)
		assert.Equal(t, "t1", response.Name)
		assert.Equal(t, "postman/collection", response.Type_)
	})

	t.Run("Delete script positive flow", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		err := client.DeleteScript("t1", "testkube")

		// then
		assert.NoError(t, err)
	})

	t.Run("Delete script fails", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		err := client.DeleteScript("t1", "testkube")

		// then
		assert.Error(t, err)
	})

	t.Run("Delete all scripts positive flow", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		err := client.DeleteScripts("testkube")

		// then
		assert.NoError(t, err)
	})

	t.Run("Delete all scripts fails", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		err := client.DeleteScripts("testkube")

		// then
		assert.Error(t, err)
	})

	t.Run("List scripts", func(t *testing.T) {
		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintf(w, `[{"id":"1", "name":"t1", "content":{"data":"{}"}, "type":"postman/collection"},{"id":"2", "name":"t2", "content":{"data":"{}"}, "type":"cypress/project"}]`)
		}))
		defer srv.Close()

		client := NewDefaultDirectScriptsAPI()
		client.URI = srv.URL

		// when
		scripts, err := client.ListScripts("testkube", nil)

		// then
		assert.NoError(t, err)
		assert.Len(t, scripts, 2)
	})

}
