package localinstall

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecrets_CreatedOnceThenReused(t *testing.T) {
	dir := t.TempDir()

	first, err := loadOrCreateSecrets(dir)
	require.NoError(t, err)
	again, err := loadOrCreateSecrets(dir)
	require.NoError(t, err)

	assert.Equal(t, first, again)
	assert.True(t, strings.HasPrefix(first.RunnerKey, "tkckey_agent_"), first.RunnerKey)
	assert.NotEqual(t, first.MasterPassword, first.MinioPassword)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "secrets.json"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestSecrets_NeverReplacedUnderExistingData(t *testing.T) {
	tests := map[string]func(dir string){
		"data kept, passwords deleted": func(dir string) {
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "data", "testkube", "data-testkube-enterprise-postgresql-0"), 0o755))
		},
		"passwords file damaged": func(dir string) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.json"), []byte(`{"runnerKey":"tkckey_agent_x"}`), 0o600))
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			setup(dir)
			before, _ := os.ReadFile(filepath.Join(dir, "secrets.json"))

			_, err := loadOrCreateSecrets(dir)

			require.Error(t, err)
			after, _ := os.ReadFile(filepath.Join(dir, "secrets.json"))
			assert.Equal(t, before, after, "nothing written")
		})
	}
}

func TestSecrets_EmptyDataFolderIsAFreshStart(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data", "testkube"), 0o755))

	_, err := loadOrCreateSecrets(dir)

	assert.NoError(t, err)
}
