package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const executionID = "id-1"

var testEventTypes = []testkube.EventType{testkube.EventType("")}

func TestWebhookListener_Notify(t *testing.T) {

	t.Run("send event success response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event testkube.Event
			err := json.NewDecoder(r.Body).Decode(&event)
			// then
			assert.NoError(t, err)
			assert.Equal(t, executionID, event.TestExecution.Id)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		l := NewWebhookListener("l1", svr.URL, "", testEventTypes)

		// when
		r := l.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())

	})

	t.Run("send event failed response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		l := NewWebhookListener("l1", svr.URL, "", testEventTypes)

		// when
		r := l.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())

	})

	t.Run("send event bad uri", func(t *testing.T) {
		// given

		s := NewWebhookListener("l1", "http://baduri.badbadbad", "", testEventTypes)

		// when
		r := s.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())
	})

}

func exampleExecution() *testkube.Execution {
	execution := testkube.NewQueuedExecution()
	execution.Id = executionID
	return execution
}
