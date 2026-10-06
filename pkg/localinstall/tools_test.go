package localinstall

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func gzipTar(t *testing.T, name string, content []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func serveFiles(t *testing.T, files map[string][]byte) (*httptest.Server, *atomic.Int32) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server, &hits
}

func testInstaller(dir, goos, goarch string, sources map[string]toolSource) *ToolInstaller {
	return &ToolInstaller{dir: dir, goos: goos, goarch: goarch, sources: sources, client: http.DefaultClient}
}

func TestToolInstaller_ChecksumMismatchLeavesNoBinary(t *testing.T) {
	server, _ := serveFiles(t, map[string][]byte{
		"/kind-linux-amd64":        []byte("tampered"),
		"/kind-linux-amd64.sha256": []byte(sha256Hex([]byte("original")) + "  kind-linux-amd64\n"),
	})
	dir := t.TempDir()
	installer := testInstaller(dir, "linux", "amd64", map[string]toolSource{
		"kind": {url: server.URL + "/kind-%[1]s-%[2]s", checksumURL: server.URL + "/kind-%[1]s-%[2]s.sha256"},
	})

	_, err := installer.Install(context.Background(), "kind")

	assert.ErrorIs(t, err, ErrChecksumMismatch)
	entries, _ := os.ReadDir(dir)
	assert.Empty(t, entries)
}

func TestToolInstaller_ExtractsBinaryFromTarball(t *testing.T) {
	tarball := gzipTar(t, "darwin-arm64/helm", []byte("helm binary"))
	server, _ := serveFiles(t, map[string][]byte{
		"/helm-darwin-arm64.tar.gz":           tarball,
		"/helm-darwin-arm64.tar.gz.sha256sum": []byte(sha256Hex(tarball) + "  helm-darwin-arm64.tar.gz\n"),
	})
	installer := testInstaller(t.TempDir(), "darwin", "arm64", map[string]toolSource{
		"helm": {
			url:           server.URL + "/helm-%[1]s-%[2]s.tar.gz",
			checksumURL:   server.URL + "/helm-%[1]s-%[2]s.tar.gz.sha256sum",
			pathInTarball: "%[1]s-%[2]s/helm",
		},
	})

	path, err := installer.Install(context.Background(), "helm")

	require.NoError(t, err)
	data, _ := os.ReadFile(path)
	assert.Equal(t, "helm binary", string(data))
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	}
}

func TestToolInstaller_UnsupportedPlatformDownloadsNothing(t *testing.T) {
	for _, platform := range [][2]string{{"windows", "amd64"}, {"linux", "386"}} {
		server, hits := serveFiles(t, nil)
		installer := testInstaller(t.TempDir(), platform[0], platform[1], map[string]toolSource{
			"kind": {url: server.URL + "/kind", checksumURL: server.URL + "/kind.sha256"},
		})

		_, err := installer.Install(context.Background(), "kind")

		assert.ErrorIs(t, err, ErrUnsupportedPlatform, "%v", platform)
		assert.Zero(t, hits.Load(), "%v", platform)
	}
}

func TestAddToolsDirToPath_UserToolsStillWin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs an executable without an .exe suffix")
	}
	home, userBin := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", userBin)
	ours := filepath.Join(home, ".testkube", "bin")
	require.NoError(t, os.MkdirAll(ours, 0o755))
	for _, dir := range []string{userBin, ours} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), nil, 0o755))
	}

	require.NoError(t, AddToolsDirToPath())

	found, err := exec.LookPath("kubectl")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(userBin, "kubectl"), found)
}
