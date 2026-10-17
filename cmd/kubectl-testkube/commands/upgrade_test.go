package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
)

func TestUpgradeCmd_ChangedFlags(t *testing.T) {
	cmd := NewUpgradeCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--reuse-values", "--no-postgres=false"}))

	changed := changedFlags(cmd)

	assert.True(t, changed["reuse-values"])
	assert.True(t, changed["no-postgres"])
	assert.False(t, changed["no-mongo"], "a flag left at its default must not override a reused value")
}

// A misspelled name would never count as passed, and its setting would never be sent.
func TestUpgradeCmd_DefinesTheFlagsOfReusedSettings(t *testing.T) {
	cmd := NewUpgradeCmd()

	// --multi-namespace and --no-crds are defined only by `pro init`, so an upgrade with
	// --reuse-values always keeps the release's values for them.
	flags := append([]string{"no-minio", "no-mongo", "no-postgres", "embedded-nats"}, common.RunnerEndpointFlags()...)
	for _, name := range flags {
		assert.NotNil(t, cmd.Flags().Lookup(name), "flag %q", name)
	}
}
