package localinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveFiles_KeepsDataToolsReportsAndConfig(t *testing.T) {
	dir := t.TempDir()
	mk := func(rel string) {
		p := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
		require.NoError(t, os.WriteFile(p, []byte("x"), 0o600))
	}
	gone := []string{"kind.yaml", "kubeconfig", "helm/cache/index.yaml", "install-123/private.yaml", "cluster-id"}
	kept := []string{"data/testkube/pvc/data/f", "bin/kind", "logs/install-1.txt", "secrets.json", "config.json", "github-app.pem"}
	for _, rel := range append(gone, kept...) {
		mk(rel)
	}

	removed, err := removeFiles(dir)

	require.NoError(t, err)
	assert.True(t, removed)
	for _, rel := range gone {
		assert.NoFileExists(t, filepath.Join(dir, rel))
	}
	for _, rel := range kept {
		assert.FileExists(t, filepath.Join(dir, rel))
	}
}

func TestRemoveFiles_NothingThereIsNotAnError(t *testing.T) {
	removed, err := removeFiles(t.TempDir())

	assert.NoError(t, err)
	assert.False(t, removed)
}

func dataHome(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), ".testkube")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data", "testkube", "pvc", "data"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data", "testkube", "pvc", "data", "pg"), []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.json"), []byte("{}"), 0o600))
	return dir
}

func TestDeleteData_ThroughAContainerThenThePasswords(t *testing.T) {
	dir := dataHome(t)
	var calls [][]string
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}

	removed, sudo, err := deleteData(context.Background(), dir, run)

	require.NoError(t, err)
	assert.True(t, removed)
	assert.Empty(t, sudo)
	assert.NoDirExists(t, filepath.Join(dir, "data"))
	assert.NoFileExists(t, filepath.Join(dir, "secrets.json"))
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0], "--pull=never", "no download just to delete")
	assert.Contains(t, calls[0], "/d/testkube")
}

func TestDeleteData_NeverThroughASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges")
	}
	dir := filepath.Join(t.TempDir(), ".testkube")
	elsewhere := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(elsewhere, "precious"), []byte("x"), 0o600))
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.Symlink(elsewhere, filepath.Join(dir, "data")))

	_, _, err := deleteData(context.Background(), dir, nil)

	assert.ErrorIs(t, err, ErrDataUnsafe)
	assert.FileExists(t, filepath.Join(elsewhere, "precious"))
}

func TestDeleteData_StuckFilesKeepThePasswordsAndOfferSudo(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("needs a file this user can't delete")
	}
	dir := dataHome(t)
	locked := filepath.Join(dir, "data", "testkube", "pvc")
	require.NoError(t, os.Chmod(locked, 0o500))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	containerFails := func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("no image") }

	removed, sudo, err := deleteData(context.Background(), dir, containerFails)

	assert.Error(t, err)
	assert.False(t, removed)
	assert.Equal(t, "sudo rm -rf '"+filepath.Join(dir, "data")+"'", sudo)
	assert.FileExists(t, filepath.Join(dir, "secrets.json"), "kept while data remains")
}
