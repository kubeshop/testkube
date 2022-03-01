package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

const executionID = "id-1"

func TestWebhook(t *testing.T) {

	t.Run("send event success response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event testkube.WebhookEvent
			err := json.NewDecoder(r.Body).Decode(&event)
			// then
			assert.NoError(t, err)
			assert.Equal(t, executionID, event.Execution.Id)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		s := NewServer()
		s.RunWorkers()

		execution := testkube.NewQueuedExecution()
		execution.Id = executionID

		// when
		s.Send(testkube.WebhookEvent{
			Type_:     testkube.WebhookTypeStartTest,
			Uri:       svr.URL,
			Execution: execution,
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

		s := NewServer()
		s.RunWorkers()

		execution := testkube.NewQueuedExecution()
		execution.Id = executionID

		// when
		s.Send(testkube.WebhookEvent{
			Type_:     testkube.WebhookTypeStartTest,
			Uri:       svr.URL,
			Execution: execution,
		})

		// then
		r := <-s.Responses
		assert.Equal(t, http.StatusBadGateway, r.Response.StatusCode)

	})

}
