package runner

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
)

type uploadAttempt struct {
	contentLength int64
	bodyLen       int
}

// uploadTestServer records every PUT and runs the per-attempt handler to decide
// the response. handler returns the HTTP status to send; a status of 0 means
// "hijack and drop the connection" to simulate a broken transport.
type uploadTestServer struct {
	mu       sync.Mutex
	attempts []uploadAttempt
	statusFn func(n int) int
}

func newUploadServer(statusFn func(n int) int) (*httptest.Server, *uploadTestServer) {
	rec := &uploadTestServer{statusFn: statusFn}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		n := len(rec.attempts) + 1
		rec.mu.Unlock()

		status := rec.statusFn(n)
		if status == 0 {
			// Read part of the body, then drop the connection mid-transfer.
			buf := make([]byte, 4)
			_, _ = io.ReadFull(r.Body, buf)
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, err := hj.Hijack()
				if err == nil {
					_ = conn.Close()
				}
			}
			rec.mu.Lock()
			rec.attempts = append(rec.attempts, uploadAttempt{contentLength: r.ContentLength, bodyLen: len(buf)})
			rec.mu.Unlock()
			return
		}

		body, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.attempts = append(rec.attempts, uploadAttempt{contentLength: r.ContentLength, bodyLen: len(body)})
		rec.mu.Unlock()
		w.WriteHeader(status)
	}))
	return srv, rec
}

func (r *uploadTestServer) snapshot() []uploadAttempt {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]uploadAttempt{}, r.attempts...)
}

func newLogWriter(t *testing.T, url string) ExecutionLogsWriter {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockClient := controlplaneclient.NewMockClient(ctrl)
	// Save() fetches a fresh presigned URL on every attempt, so allow any count.
	mockClient.EXPECT().
		SaveExecutionLogsGetPresignedURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(url, nil).
		AnyTimes()

	w, err := NewExecutionLogsWriter(mockClient, "env-1", "exec-1", "wf-1", true, false)
	require.NoError(t, err)
	return w
}

// saveWithRetry re-calls Save() on failure, mirroring how the runner retries
// the final save, so the rewind-on-retry behavior can be exercised.
func saveWithRetry(w ExecutionLogsWriter, maxAttempts int) error {
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = w.Save(context.Background())
		if err == nil {
			return nil
		}
	}
	return err
}

// A transient first failure must recover on retry with the full body.
func TestExecutionLogsWriter_RetryResendsFullBody(t *testing.T) {
	const payload = "hello-testkube-execution-log-payload"
	srv, rec := newUploadServer(func(n int) int {
		if n == 1 {
			return http.StatusInternalServerError
		}
		return http.StatusOK
	})
	defer srv.Close()

	w := newLogWriter(t, srv.URL)
	_, err := w.Write([]byte(payload))
	require.NoError(t, err)

	require.NoError(t, saveWithRetry(w, 5))

	attempts := rec.snapshot()
	require.GreaterOrEqual(t, len(attempts), 2)
	assert.Equal(t, int64(len(payload)), attempts[0].contentLength)
	assert.Equal(t, len(payload), attempts[0].bodyLen, "attempt 1 carries the full body")
	assert.Equal(t, int64(len(payload)), attempts[1].contentLength)
	assert.Equal(t, len(payload), attempts[1].bodyLen, "retry resends the full body, not 0 bytes")
}

// Mid-file geometry: the first PUT drops the connection mid-transfer (the
// customer's reported case), leaving the offset partway through the file.
// Rewind must still let the retry send the full body.
func TestExecutionLogsWriter_RetryRecoversFromDroppedConnection(t *testing.T) {
	// Payload large enough that a partial transfer is plausible before the drop.
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte('a' + (i % 26))
	}
	srv, rec := newUploadServer(func(n int) int {
		if n == 1 {
			return 0 // hijack + close -> transport error
		}
		return http.StatusOK
	})
	defer srv.Close()

	w := newLogWriter(t, srv.URL)
	_, err := w.Write(payload)
	require.NoError(t, err)

	require.NoError(t, saveWithRetry(w, 5))

	attempts := rec.snapshot()
	require.GreaterOrEqual(t, len(attempts), 2)
	last := attempts[len(attempts)-1]
	assert.Equal(t, int64(len(payload)), last.contentLength)
	assert.Equal(t, len(payload), last.bodyLen, "the successful retry sends the full body after a dropped connection")
}

// An empty log uploads with http.NoBody and succeeds on the first attempt (R3).
func TestExecutionLogsWriter_EmptyLog(t *testing.T) {
	srv, rec := newUploadServer(func(n int) int { return http.StatusOK })
	defer srv.Close()

	w := newLogWriter(t, srv.URL)
	require.NoError(t, saveWithRetry(w, 5))

	attempts := rec.snapshot()
	require.Len(t, attempts, 1)
	assert.Equal(t, int64(0), attempts[0].contentLength)
	assert.Equal(t, 0, attempts[0].bodyLen)
}

// Repeated transient failures still preserve the body on a later attempt.
func TestExecutionLogsWriter_RepeatedFailuresPreserveBody(t *testing.T) {
	const payload = "another-log-payload-for-repeated-failures"
	srv, rec := newUploadServer(func(n int) int {
		if n <= 2 {
			return http.StatusInternalServerError
		}
		return http.StatusOK
	})
	defer srv.Close()

	w := newLogWriter(t, srv.URL)
	_, err := w.Write([]byte(payload))
	require.NoError(t, err)

	require.NoError(t, saveWithRetry(w, 5))

	attempts := rec.snapshot()
	require.GreaterOrEqual(t, len(attempts), 3)
	third := attempts[2]
	assert.Equal(t, int64(len(payload)), third.contentLength)
	assert.Equal(t, len(payload), third.bodyLen, "third attempt still carries the full body")
}
