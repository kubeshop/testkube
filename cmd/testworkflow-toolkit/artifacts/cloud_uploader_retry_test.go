package artifacts

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func withShortUploadBackoff(t *testing.T) {
	t.Helper()
	old := uploadRetryBaseDelay
	uploadRetryBaseDelay = time.Millisecond
	t.Cleanup(func() { uploadRetryBaseDelay = old })
}

// A brief 5xx blip on the signed URL should be retried transparently: the
// body is a *bytes.Reader (io.Seeker), so putObject rewinds and replays.
func TestCloudUploader_putObject_RetriesOn5xxThenSucceeds(t *testing.T) {
	withShortUploadBackoff(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u := &cloudUploader{}
	body := bytes.NewReader([]byte("hello"))
	err := u.putObject(server.URL, "artifact.txt", body, int64(body.Len()))

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

// 4xx signals a caller / URL problem; retrying spams the signed URL with a
// request that will keep failing. Loop must short-circuit after one attempt.
func TestCloudUploader_putObject_DoesNotRetryOn4xx(t *testing.T) {
	withShortUploadBackoff(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	u := &cloudUploader{}
	body := bytes.NewReader([]byte("hello"))
	err := u.putObject(server.URL, "artifact.txt", body, int64(body.Len()))

	assert.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

// A zero-byte artifact has no body to rewind, so the retry loop must not
// short-circuit on it. Empty PUTs are inherently replayable; 5xx blips should
// still be recovered.
func TestCloudUploader_putObject_RetriesZeroByteBodyOn5xx(t *testing.T) {
	withShortUploadBackoff(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u := &cloudUploader{}
	err := u.putObject(server.URL, "empty.txt", bytes.NewReader(nil), 0)

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

// Exhausting the retry budget must surface an error so the caller sets the
// error flag on the uploader (visible as "failed" in the artifact summary).
func TestCloudUploader_putObject_GivesUpAfterRetryBudget(t *testing.T) {
	withShortUploadBackoff(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	u := &cloudUploader{}
	body := bytes.NewReader([]byte("hello"))
	err := u.putObject(server.URL, "artifact.txt", body, int64(body.Len()))

	assert.Error(t, err)
	assert.Equal(t, int32(uploadRetryCount), atomic.LoadInt32(&calls))
}
