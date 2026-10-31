package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileOrStdin(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(`{"text": "piped"}`))

	t.Run("reads standard input for -", func(t *testing.T) {
		b, err := ReadFileOrStdin(cmd, "-")

		require.NoError(t, err)
		assert.Equal(t, `{"text": "piped"}`, string(b))
	})

	t.Run("reads a file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "payload.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"text": "file"}`), 0o600))

		b, err := ReadFileOrStdin(cmd, path)

		require.NoError(t, err)
		assert.Equal(t, `{"text": "file"}`, string(b))
	})

	t.Run("fails for a missing file", func(t *testing.T) {
		_, err := ReadFileOrStdin(cmd, filepath.Join(t.TempDir(), "missing.json"))

		assert.Error(t, err)
	})
}
