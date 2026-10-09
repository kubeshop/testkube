package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestNewDirectAPIClientStreamsNotificationsWithSSEClient(t *testing.T) {
	const pause = 300 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"log\":\"first\"}\n\n")
		flusher.Flush()
		time.Sleep(pause)
		fmt.Fprint(w, "data: {\"log\":\"second\"}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	// The regular client would cut the stream before the second notification
	httpClient := &http.Client{Timeout: pause / 3}
	sseClient := &http.Client{}
	client := NewDirectAPIClient(httpClient, sseClient, server.URL, "")

	notifications, err := client.GetTestWorkflowExecutionNotifications("execution-id")
	require.NoError(t, err)

	var received []testkube.TestWorkflowExecutionNotification
	timeout := time.After(5 * time.Second)
	for done := false; !done; {
		select {
		case n, ok := <-notifications:
			if !ok {
				done = true
				break
			}
			received = append(received, n)
		case <-timeout:
			t.Fatal("notification stream did not finish")
		}
	}

	require.Len(t, received, 2)
	assert.Equal(t, "second", received[1].Log)
}
