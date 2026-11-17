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

// uploadRecorder captures what the test server saw across attempts: how many
// requests arrived and the body of the last one it served.
type uploadRecorder struct {
	mu       sync.Mutex
	calls    int32
	received []byte
}

func (r *uploadRecorder) attempts() int32 { return atomic.LoadInt32(&r.calls) }

func (r *uploadRecorder) body() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.received
}

// dropFirstNThenServe hijacks and closes the connection for the first n requests
// (a transport-level failure that makes net/http close the request body), then
// reads the full body and returns 200. Assertions use t.Errorf, not require.*,
// because this runs in the server's handler goroutine, where FailNow is invalid.
func dropFirstNThenServe(t *testing.T, n int32, rec *uploadRecorder) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&rec.calls, 1) <= n {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Errorf("test server does not support hijacking")
				http.Error(w, "no hijack", http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Errorf("hijack failed: %v", err)
				return
			}
			_ = conn.Close()
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		rec.mu.Lock()
		rec.received = body
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}
}

// TKC-7667: a transient first-attempt failure closes the *os.File body, so the
// retry must rewind and resend the full file rather than failing with
// "cannot rewind body for retry: ... file already closed".
func TestCloudUploader_putObject_RecoversFileBodyAfterTransportFailure(t *testing.T) {
	withShortUploadBackoff(t)
	payload := []byte("the-full-artifact-contents-for-tkc-7667")

	rec := &uploadRecorder{}
	server := httptest.NewServer(dropFirstNThenServe(t, 1, rec))
	defer server.Close()

	f := tempFileBody(t, payload)
	err := (&cloudUploader{}).putObject(server.URL, "artifact.txt", f, int64(len(payload)))

	assert.NoError(t, err, "retry must recover after the first-attempt transport failure")
	assert.GreaterOrEqual(t, rec.attempts(), int32(2), "a retry must have happened")
	assert.Equal(t, payload, rec.body(), "the retry must resend the full file, not a truncated or empty body")
}

// Rewind must survive more than one transport failure.
func TestCloudUploader_putObject_RecoversFileBodyAfterTwoTransportFailures(t *testing.T) {
	withShortUploadBackoff(t)
	payload := []byte("payload-that-must-survive-two-retries")

	rec := &uploadRecorder{}
	server := httptest.NewServer(dropFirstNThenServe(t, 2, rec))
	defer server.Close()

	f := tempFileBody(t, payload)
	err := (&cloudUploader{}).putObject(server.URL, "artifact.txt", f, int64(len(payload)))

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, rec.attempts(), int32(3))
	assert.Equal(t, payload, rec.body(), "third attempt must still carry the full file")
}
