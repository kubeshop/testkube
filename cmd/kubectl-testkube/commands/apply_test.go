package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDryRunSkipsTheConnectionHooks(t *testing.T) {
	calls := 0
	hook := unlessDryRun(func(*cobra.Command, []string) { calls++ })

	cmd := NewApplyCmd()
	hook(cmd, nil)
	assert.Equal(t, 1, calls, "an apply connects")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))
	hook(cmd, nil)
	assert.Equal(t, 1, calls, "a dry run does not")
}
