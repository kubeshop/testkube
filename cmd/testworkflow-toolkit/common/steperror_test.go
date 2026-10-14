package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
)

func TestWriteStepError(t *testing.T) {
	tests := []struct {
		name     string
		existing *string
	}{
		{
			name: "writes the message to the file that TK_ERR_FILE names",
		},
		{
			name:     "replaces a longer message that an earlier write left in the file",
			existing: common.Ptr("cannot fetch the artifacts of an earlier attempt: connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "error")
			if tt.existing != nil {
				require.NoError(t, os.WriteFile(path, []byte(*tt.existing), 0666))
			}
			t.Setenv(constants.EnvStepErrorFile, path)

			writeStepError("error cloning repository: exit status 128")

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "error cloning repository: exit status 128", string(content))
		})
	}
}
