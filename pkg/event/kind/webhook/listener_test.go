package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const executionID = "id-1"

var testEventTypes = []testkube.EventType{testkube.EventType("")}

func TestWebhookListener_Notify(t *testing.T) {
	t.Parallel()
	t.Run("send event success response", func(t *testing.T) {
		t.Parallel()
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

		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil)

		// when
		r := l.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())

	})

	t.Run("send event failed response", func(t *testing.T) {
		t.Parallel()
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil)

		// when
		r := l.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())

	})

	t.Run("send event bad uri", func(t *testing.T) {
		t.Parallel()
		// given

		s := NewWebhookListener("l1", "http://baduri.badbadbad", "", testEventTypes, "", "", nil)

		// when
		r := s.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())
	})

	t.Run("send event success response using payload field", func(t *testing.T) {
		t.Parallel()
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := bytes.NewBuffer([]byte{})
			err := json.NewEncoder(body).Encode(testkube.Event{
				Type_:         testkube.EventStartTest,
				TestExecution: exampleExecution(),
			})
			assert.NoError(t, err)

			data := make(map[string]string, 0)
			err = json.NewDecoder(r.Body).Decode(&data)
			// then
			assert.NoError(t, err)
			assert.Equal(t, string(body.Bytes()), data["field"])
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "field", "", nil)

		// when
		r := l.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())

	})

	t.Run("send event success response using payload template", func(t *testing.T) {
		t.Parallel()
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			// then
			assert.Equal(t, "{\"id\": \"12345\"}", string(body))
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "{\"id\": \"{{ .Id }}\"}", map[string]string{"Content-Type": "application/json"})

		// when
		r := l.Notify(testkube.Event{
			Id:            "12345",
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())

	})
}

func exampleExecution() *testkube.Execution {
	execution := testkube.NewQueuedExecution()
	execution.Id = executionID
	return execution
}
