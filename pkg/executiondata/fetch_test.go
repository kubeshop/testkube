package executiondata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestFetchArtifacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content of " + r.URL.Path))
	}))
	defer server.Close()

	t.Run("writes artifacts preserving their layout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"results/**"}).Return([]Artifact{
			{Path: "results/summary.json", Url: server.URL + "/a"},
			{Path: "results/nested/report.xml", Url: server.URL + "/b"},
		}, nil)

		dir := t.TempDir()
		result, err := FetchArtifacts(context.Background(), repository, nil, "exec-1", []string{"results/**"}, dir)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Files)
		assert.Equal(t, int64(len("content of /a")+len("content of /b")), result.Bytes)

		content, err := os.ReadFile(filepath.Join(dir, "results", "summary.json"))
		require.NoError(t, err)
		assert.Equal(t, "content of /a", string(content))

		content, err = os.ReadFile(filepath.Join(dir, "results", "nested", "report.xml"))
		require.NoError(t, err)
		assert.Equal(t, "content of /b", string(content))
	})

	t.Run("refuses artifacts escaping the target directory", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", gomock.Any()).
			Return([]Artifact{{Path: "../../etc/passwd", Url: server.URL + "/a"}}, nil)

		dir := t.TempDir()
		_, err := FetchArtifacts(context.Background(), repository, nil, "exec-1", nil, dir)
		assert.ErrorContains(t, err, "would be written outside")
	})

	t.Run("reports a missing target directory", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		_, err := FetchArtifacts(context.Background(), NewMockExecutionRepository(ctrl), nil, "exec-1", nil, "")
		assert.ErrorContains(t, err, "no target directory")
	})

	t.Run("explains a missing control plane connection", func(t *testing.T) {
		_, err := FetchArtifacts(context.Background(), nil, nil, "exec-1", nil, t.TempDir())
		assert.ErrorContains(t, err, "no connection to the control plane")
	})
}

func TestTargetPath(t *testing.T) {
	dir := filepath.Join("tmp", "fetch")

	target, err := targetPath(dir, "results/summary.json")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "results", "summary.json"), target)

	target, err = targetPath(dir+string(os.PathSeparator), "a.txt")
	require.NoError(t, err, "a trailing separator on the target directory must not matter")
	assert.Equal(t, filepath.Join(dir, "a.txt"), target)

	_, err = targetPath(dir, "../escape.txt")
	assert.Error(t, err)

	_, err = targetPath(dir, "")
	assert.Error(t, err, "an empty path resolves to the directory itself, which is not a file")
}
