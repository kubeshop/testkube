package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeCmd_ChangedFlags(t *testing.T) {
	cmd := NewUpgradeCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--reuse-values", "--no-postgres=false"}))

	changed := changedFlags(cmd)

	assert.True(t, changed["reuse-values"])
	assert.True(t, changed["no-postgres"])
	assert.False(t, changed["no-mongo"], "a flag left at its default must not override a reused value")
}
