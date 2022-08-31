package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

const executionID = "id-1"

func TestWebhook(t *testing.T) {

	t.Run("send event success response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event testkube.Event
			err := json.NewDecoder(r.Body).Decode(&event)
			// then
			assert.NoError(t, err)
			assert.Equal(t, executionID, event.Execution.Id)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		s := NewSimpleEmitter()
		s.RunWorkers()

		// when
		s.sendHttpEvent(testkube.Event{
			Type_:     testkube.EventStartTest,
			Uri:       svr.URL,
			Execution: exampleExecution(),
		})

		// then
		r := <-s.Responses
		assert.Equal(t, 200, r.Response.StatusCode)

	})

	t.Run("send event failed response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		s := NewSimpleEmitter()
		s.RunWorkers()

		// when
		s.sendHttpEvent(testkube.Event{
			Type_:     testkube.EventStartTest,
			Uri:       svr.URL,
			Execution: exampleExecution(),
		})

		// then
		r := <-s.Responses
		assert.Equal(t, http.StatusBadGateway, r.Response.StatusCode)

	})

	t.Run("send event bad uri", func(t *testing.T) {
		// given
		s := NewSimpleEmitter()
		s.RunWorkers()

		// when
		s.sendHttpEvent(testkube.Event{
			Type_:     testkube.EventStartTest,
			Uri:       "http://baduri.badbadbad",
			Execution: exampleExecution(),
		})

		// then
		r := <-s.Responses
		assert.Error(t, r.Error)
	})

}

func exampleExecution() *testkube.Execution {
	execution := testkube.NewQueuedExecution()
	execution.Id = executionID
	return execution
}

func NewSimpleEmitter() *Emitter {
	return NewEmitter(&executorsclientv1.WebhooksClient{})

}
