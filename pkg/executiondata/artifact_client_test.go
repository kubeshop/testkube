package executiondata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewArtifactClient(t *testing.T) {
	t.Run("verifies by default", func(t *testing.T) {
		client := NewArtifactClient(false)
		assert.Equal(t, ArtifactDownloadTimeout, client.Timeout)
		assert.Nil(t, client.Transport, "the default transport verifies, so there is nothing to override")
	})

	t.Run("skips verification when the deployment asked for it", func(t *testing.T) {
		client := NewArtifactClient(true)
		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.TLSClientConfig)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("leaves the shared transport alone", func(t *testing.T) {
		// Configuring the process-wide transport instead of a clone would make every
		// other client in this process skip verification too.
		NewArtifactClient(true)

		shared, ok := http.DefaultTransport.(*http.Transport)
		require.True(t, ok)
		if shared.TLSClientConfig != nil {
			assert.False(t, shared.TLSClientConfig.InsecureSkipVerify)
		}
	})
}

// Object storage in a self-hosted deployment presents a certificate signed by a private
// CA, which this process has no reason to trust. Reading an artifact from it has to
// follow the same decision the rest of the worker makes about storage certificates,
// otherwise a workflow cannot download what it just uploaded.
func TestReadArtifactOverUntrustedTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"cases":3}`))
	}))
	defer server.Close()

	artifacts := []Artifact{{Path: "fixtures/data.json", Url: server.URL + "/fixtures/data.json", Size: 11}}

	t.Run("reads it when verification is skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"fixtures/data.json"}).Return(artifacts, nil)

		content, err := ReadArtifact(context.Background(), repository, NewArtifactClient(true), "exec-1", "fixtures/data.json")
		require.NoError(t, err)
		assert.Equal(t, `{"cases":3}`, content)
	})

	t.Run("refuses the certificate otherwise", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"fixtures/data.json"}).Return(artifacts, nil)

		_, err := ReadArtifact(context.Background(), repository, nil, "exec-1", "fixtures/data.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate")
	})
}

func TestFetchArtifactsOverUntrustedTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"passed":true}`))
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	repository := NewMockExecutionRepository(ctrl)
	repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"results/**"}).
		Return([]Artifact{{Path: "results/summary.json", Url: server.URL + "/results/summary.json", Size: 15}}, nil)

	result, err := FetchArtifacts(context.Background(), repository, NewArtifactClient(true), "exec-1", []string{"results/**"}, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Files)
}
