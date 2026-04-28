package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestDirectClientGetTestWorkflowExecutionNotificationsUsesResumeQueryAndContext(t *testing.T) {
	requests := make(chan *http.Request, 1)
	requestDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ": connected\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(requestDone)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	notifications := make(chan testkube.TestWorkflowExecutionNotification)
	client := NewDirectClient[testkube.TestWorkflow](server.Client(), server.URL, "").WithSSEClient(server.Client())

	err := client.GetTestWorkflowExecutionNotifications(server.URL+"/notifications", notifications, TestWorkflowExecutionNotificationsOptions{
		Context:          ctx,
		ResumeAfterSeqNo: 42,
		StreamID:         "stream-123",
	})
	require.NoError(t, err)

	select {
	case req := <-requests:
		assert.Equal(t, "42", req.URL.Query().Get("resumeAfterSeqNo"))
		assert.Equal(t, "stream-123", req.URL.Query().Get("streamId"))
		assert.Equal(t, "text/event-stream", req.Header.Get("Accept"))
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE request")
	}

	cancel()

	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE request context cancellation")
	}

	select {
	case _, ok := <-notifications:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notifications channel to close")
	}
}
