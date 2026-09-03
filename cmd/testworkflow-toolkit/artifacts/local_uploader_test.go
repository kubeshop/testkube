package artifacts

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/testworkflows/localartifacts"
)

func TestLocalUploaderPrefixesSafePathAndAuthenticatesRequest(t *testing.T) {
	var receivedPath, receivedToken string
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, http.MethodPut, request.Method)
		receivedPath = request.Header.Get(localartifacts.PathHeader)
		receivedToken = request.Header.Get(localartifacts.TokenHeader)
		var err error
		receivedBody, err = io.ReadAll(request.Body)
		require.NoError(t, err)
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	uploader, err := NewLocalUploader(server.URL+"/upload", "relay-token", "step-7")
	require.NoError(t, err)
	require.NoError(t, uploader.Start())
	require.NoError(t, uploader.Add("test-results/junit.xml", io.NopCloser(strings.NewReader("<xml/>")), int64(len("<xml/>"))))
	require.NoError(t, uploader.End())

	require.Equal(t, "steps/step-7/test-results/junit.xml", receivedPath)
	require.Equal(t, "relay-token", receivedToken)
	require.Equal(t, []byte("<xml/>"), receivedBody)
}

func TestLocalUploaderRejectsUnsafeInputs(t *testing.T) {
	_, err := NewLocalUploader("ftp://relay", "relay-token", "step-1")
	require.ErrorContains(t, err, "HTTP")

	_, err = NewLocalUploader("http://relay/upload", "", "step-1")
	require.ErrorContains(t, err, "token")

	_, err = NewLocalUploader("http://relay/upload", "relay-token", "../step")
	require.ErrorContains(t, err, "step reference")

	uploader, err := NewLocalUploader("http://relay/upload", "relay-token", "step-1")
	require.NoError(t, err)
	err = uploader.Add("../escape.txt", io.NopCloser(strings.NewReader("content")), int64(len("content")))
	require.ErrorContains(t, err, "unsafe")
}

func TestLocalUploaderDoesNotExposeTokenInReceiverErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "receiver rejected upload", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	uploader, err := NewLocalUploader(server.URL+"/upload", "sensitive-relay-token", "step-1")
	require.NoError(t, err)
	err = uploader.Add("result.txt", io.NopCloser(strings.NewReader("content")), int64(len("content")))
	require.Error(t, err)
	require.NotContains(t, err.Error(), "sensitive-relay-token")
}
