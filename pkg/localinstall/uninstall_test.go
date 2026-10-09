package localinstall

import (
	"os"
	"path/filepath"
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
