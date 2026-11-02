package artifacts

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tempFileBody writes payload to a temp *os.File positioned at the start.
// *os.File is an io.ReadSeekCloser, matching the real artifact body that
// triggered TKC-7667: net/http would otherwise close it between retry attempts.
func tempFileBody(t *testing.T, payload []byte) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "artifact")
	require.NoError(t, err)
	_, err = f.Write(payload)
	require.NoError(t, err)
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// dropFirstNConnections returns a handler that hijacks and closes the connection
// for the first n requests (a transport-level failure that makes net/http close
// the request body), then reads the full body and returns 200, recording it.
func dropFirstNConnections(t *testing.T, n int32, received *[]byte, mu *sync.Mutex, calls *int32) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(calls, 1) <= n {
			hj, ok := w.(http.Hijacker)
			require.True(t, ok)
			conn, _, err := hj.Hijack()
			require.NoError(t, err)
			_ = conn.Close()
			return
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		*received = body
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}
}

// TKC-7667: a transient first-attempt failure closes the *os.File body, so the
// retry must rewind and resend the full file rather than failing with
// "cannot rewind body for retry: ... file already closed".
func TestCloudUploader_putObject_RecoversFileBodyAfterTransportFailure(t *testing.T) {
	withShortUploadBackoff(t)
	payload := []byte("the-full-artifact-contents-for-tkc-7667")

	var calls int32
	var mu sync.Mutex
	var received []byte
	server := httptest.NewServer(dropFirstNConnections(t, 1, &received, &mu, &calls))
	defer server.Close()

	u := &cloudUploader{}
	f := tempFileBody(t, payload)
	err := u.putObject(server.URL, "artifact.txt", f, int64(len(payload)))

	require.NoError(t, err, "retry must recover after the first-attempt transport failure")
	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2), "a retry must have happened")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, payload, received, "the retry must resend the full file, not a truncated or empty body")
}

// Rewind must survive more than one transport failure.
func TestCloudUploader_putObject_RecoversFileBodyAfterTwoTransportFailures(t *testing.T) {
	withShortUploadBackoff(t)
	payload := []byte("payload-that-must-survive-two-retries")

	var calls int32
	var mu sync.Mutex
	var received []byte
	server := httptest.NewServer(dropFirstNConnections(t, 2, &received, &mu, &calls))
	defer server.Close()

	u := &cloudUploader{}
	f := tempFileBody(t, payload)
	err := u.putObject(server.URL, "artifact.txt", f, int64(len(payload)))

	require.NoError(t, err)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(3))
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, payload, received, "third attempt must still carry the full file")
}
