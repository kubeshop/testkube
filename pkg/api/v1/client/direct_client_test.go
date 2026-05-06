package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDirectClientResponseError_WrapsHTTPStatusError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		wantHTTPErr    bool
		wantStatusCode int
		wantIsNotFound bool
	}{
		{
			name:           "404 surfaces HTTPStatusError and IsNotFound",
			statusCode:     http.StatusNotFound,
			body:           `{"title":"Not Found","detail":"resource not found","status":404}`,
			wantHTTPErr:    true,
			wantStatusCode: http.StatusNotFound,
			wantIsNotFound: true,
		},
		{
			name:           "400 surfaces HTTPStatusError but not IsNotFound",
			statusCode:     http.StatusBadRequest,
			body:           `{"title":"Bad Request","detail":"invalid input","status":400}`,
			wantHTTPErr:    true,
			wantStatusCode: http.StatusBadRequest,
			wantIsNotFound: false,
		},
		{
			name:           "500 surfaces HTTPStatusError but not IsNotFound",
			statusCode:     http.StatusInternalServerError,
			body:           `{"title":"Internal Server Error","detail":"server failure","status":500}`,
			wantHTTPErr:    true,
			wantStatusCode: http.StatusInternalServerError,
			wantIsNotFound: false,
		},
		{
			name:           "200 returns nil",
			statusCode:     http.StatusOK,
			body:           `{}`,
			wantHTTPErr:    false,
			wantIsNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewDirectClient[testkube.TestWorkflow](http.DefaultClient, "http://localhost", "")

			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			err := c.responseError(resp)

			if !tt.wantHTTPErr {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)

			var httpErr *HTTPStatusError
			require.True(t, errors.As(err, &httpErr), "expected HTTPStatusError in error chain, got: %v", err)
			assert.Equal(t, tt.wantStatusCode, httpErr.StatusCode)
			assert.Equal(t, tt.wantIsNotFound, IsNotFound(err))
		})
	}
}
