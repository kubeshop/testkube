package cdevent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var testEventTypes = []testkube.EventType{*testkube.EventStartTestWorkflow}

func TestCDEventListener_Notify(t *testing.T) {
	t.Run("send event success response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := cloudevents.NewEventFromHTTPRequest(r)
			// then
			assert.NoError(t, err)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		client, err := cloudevents.NewClientHTTP(cloudevents.WithTarget(svr.URL))
		assert.NoError(t, err)

		l := NewCDEventListener("cdeli", "", "clusterID", "", "", testEventTypes, client)

		// when
		r := l.Notify(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
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

		client, err := cloudevents.NewClientHTTP(cloudevents.WithTarget(svr.URL))
		assert.NoError(t, err)

		l := NewCDEventListener("cdeli", "", "clusterID", "", "", testEventTypes, client)

		// when
		r := l.Notify(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())
	})

	t.Run("send event bad uri", func(t *testing.T) {
		// given

		client, err := cloudevents.NewClientHTTP(cloudevents.WithTarget("abcdef"))
		assert.NoError(t, err)

		l := NewCDEventListener("cdeli", "", "clusterID", "", "", testEventTypes, client)

		// when
		r := l.Notify(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())
	})
}

func exampleExecution() *testkube.TestWorkflowExecution {
	namespace := "testkube"

	execution := testkube.NewQueuedExecution()
	execution.Id = "1"
	execution.Name = "test-1"
	execution.Namespace = namespace
	execution.Workflow = &testkube.TestWorkflow{Name: "test", Namespace: namespace}
	return execution
}
