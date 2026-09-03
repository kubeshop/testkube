package artifacts

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/testworkflows/localartifacts"
)

func TestLocalReceiverStoresAuthenticatedUploadsAtomically(t *testing.T) {
	root := t.TempDir()
	receiver, err := NewLocalReceiver(root, "relay-token", 64)
	require.NoError(t, err)
	server := httptest.NewServer(receiver.Handler())
	t.Cleanup(server.Close)

	health, err := http.Get(server.URL + "/healthz")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, health.StatusCode)
	require.NoError(t, health.Body.Close())

	response := uploadToLocalReceiver(t, server.URL, "relay-token", "steps/step-1/test-results/result.json", []byte(`{"passed":true}`))
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.NoError(t, response.Body.Close())

	content, err := os.ReadFile(filepath.Join(root, "steps", "step-1", "test-results", "result.json"))
	require.NoError(t, err)
	require.Equal(t, []byte(`{"passed":true}`), content)
	info, err := os.Stat(filepath.Join(root, "steps", "step-1", "test-results", "result.json"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestLocalReceiverRejectsUnauthenticatedAndUnsafeUploads(t *testing.T) {
	root := t.TempDir()
	receiver, err := NewLocalReceiver(root, "relay-token", 64)
	require.NoError(t, err)
	server := httptest.NewServer(receiver.Handler())
	t.Cleanup(server.Close)

	unauthenticated := uploadToLocalReceiver(t, server.URL, "wrong-token", "steps/step-1/result.txt", []byte("nope"))
	require.Equal(t, http.StatusUnauthorized, unauthenticated.StatusCode)
	require.NoError(t, unauthenticated.Body.Close())

	traversal := uploadToLocalReceiver(t, server.URL, "relay-token", "../escape.txt", []byte("nope"))
	require.Equal(t, http.StatusBadRequest, traversal.StatusCode)
	require.NoError(t, traversal.Body.Close())

	_, err = os.Stat(filepath.Join(filepath.Dir(root), "escape.txt"))
	require.ErrorIs(t, err, os.ErrNotExist)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestLocalReceiverEnforcesCumulativeLimitAndDoesNotLeavePartialFiles(t *testing.T) {
	root := t.TempDir()
	receiver, err := NewLocalReceiver(root, "relay-token", 5)
	require.NoError(t, err)
	server := httptest.NewServer(receiver.Handler())
	t.Cleanup(server.Close)

	first := uploadToLocalReceiver(t, server.URL, "relay-token", "steps/step-1/one.txt", []byte("abc"))
	require.Equal(t, http.StatusNoContent, first.StatusCode)
	require.NoError(t, first.Body.Close())

	tooLarge := uploadToLocalReceiver(t, server.URL, "relay-token", "steps/step-1/two.txt", []byte("def"))
	require.Equal(t, http.StatusRequestEntityTooLarge, tooLarge.StatusCode)
	require.NoError(t, tooLarge.Body.Close())
	_, err = os.Stat(filepath.Join(root, "steps", "step-1", "two.txt"))
	require.ErrorIs(t, err, os.ErrNotExist)

	duplicate := uploadToLocalReceiver(t, server.URL, "relay-token", "steps/step-1/one.txt", []byte("xx"))
	require.Equal(t, http.StatusConflict, duplicate.StatusCode)
	require.NoError(t, duplicate.Body.Close())
	content, err := os.ReadFile(filepath.Join(root, "steps", "step-1", "one.txt"))
	require.NoError(t, err)
	require.Equal(t, []byte("abc"), content)
}

func TestLocalReceiverRejectsDuplicateAuthenticationHeaders(t *testing.T) {
	receiver, err := NewLocalReceiver(t.TempDir(), "relay-token", 64)
	require.NoError(t, err)
	server := httptest.NewServer(receiver.Handler())
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodPut, server.URL+"/upload", bytes.NewReader([]byte("content")))
	require.NoError(t, err)
	req.Header.Add(localartifacts.TokenHeader, "relay-token")
	req.Header.Add(localartifacts.TokenHeader, "relay-token")
	req.Header.Set(localartifacts.PathHeader, "steps/step-1/result.txt")
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.NoError(t, response.Body.Close())
}

func uploadToLocalReceiver(t *testing.T, serverURL, token, destination string, content []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, serverURL+"/upload", bytes.NewReader(content))
	require.NoError(t, err)
	req.Header.Set(localartifacts.TokenHeader, token)
	req.Header.Set(localartifacts.PathHeader, destination)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return response
}

func TestLocalReceiverRejectsSymlinkRoot(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	require.NoError(t, os.Mkdir(target, 0o700))
	link := filepath.Join(base, "link")
	require.NoError(t, os.Symlink(target, link))

	_, err := NewLocalReceiver(link, "relay-token", 64)
	require.ErrorContains(t, err, "not a symlink")
}

func TestLocalReceiverConsumesExactLimit(t *testing.T) {
	root := t.TempDir()
	receiver, err := NewLocalReceiver(root, "relay-token", 3)
	require.NoError(t, err)
	server := httptest.NewServer(receiver.Handler())
	t.Cleanup(server.Close)

	response := uploadToLocalReceiver(t, server.URL, "relay-token", "result.txt", []byte("abc"))
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.NoError(t, response.Body.Close())

	response = uploadToLocalReceiver(t, server.URL, "relay-token", "empty.txt", nil)
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.NoError(t, response.Body.Close())

	file, err := os.Open(filepath.Join(root, "empty.txt"))
	require.NoError(t, err)
	defer file.Close()
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Empty(t, content)
}

func TestLocalReceiverRejectsMoreThanTheBoundedFileCount(t *testing.T) {
	root := t.TempDir()
	receiver, err := NewLocalReceiver(root, "relay-token", 64)
	require.NoError(t, err)
	receiver.fileCount = localartifacts.MaxArtifactFiles
	server := httptest.NewServer(receiver.Handler())
	t.Cleanup(server.Close)

	response := uploadToLocalReceiver(t, server.URL, "relay-token", "result.txt", []byte("x"))
	require.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)
	require.NoError(t, response.Body.Close())
	_, err = os.Stat(filepath.Join(root, "result.txt"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
