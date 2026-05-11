package common

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

func newSkipTLSTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("skip-tls", false, "")
	cmd.Flags().Bool("insecure", false, "")
	cmd.Flags().Bool("master-insecure", false, "")
	cmd.Flags().Bool("cloud-insecure", false, "")
	return cmd
}

func TestResolveSkipTLS(t *testing.T) {
	t.Run("skip-tls has highest precedence", func(t *testing.T) {
		cfg := &config.Data{SkipTLS: false}
		cmd := newSkipTLSTestCmd()
		err := cmd.Flags().Set("insecure", "false")
		assert.NoError(t, err)
		err = cmd.Flags().Set("skip-tls", "true")
		assert.NoError(t, err)

		assert.True(t, ResolveSkipTLS(cmd, cfg))
	})

	t.Run("alias insecure works", func(t *testing.T) {
		cfg := &config.Data{SkipTLS: false}
		cmd := newSkipTLSTestCmd()
		err := cmd.Flags().Set("insecure", "true")
		assert.NoError(t, err)

		assert.True(t, ResolveSkipTLS(cmd, cfg))
	})

	t.Run("persisted config used when no explicit flags", func(t *testing.T) {
		cfg := &config.Data{SkipTLS: true}
		cmd := newSkipTLSTestCmd()

		assert.True(t, ResolveSkipTLS(cmd, cfg))
	})

	t.Run("explicit false overrides persisted true", func(t *testing.T) {
		cfg := &config.Data{SkipTLS: true}
		cmd := newSkipTLSTestCmd()
		err := cmd.Flags().Set("skip-tls", "false")
		assert.NoError(t, err)

		assert.False(t, ResolveSkipTLS(cmd, cfg))
	})
}

func TestSyncSkipTLSFromFlags(t *testing.T) {
	cfg := &config.Data{SkipTLS: false, CloudContext: config.CloudContext{SkipTLS: false}}
	cmd := newSkipTLSTestCmd()

	err := cmd.Flags().Set("skip-tls", "true")
	assert.NoError(t, err)

	value := SyncSkipTLSFromFlags(cmd, cfg)
	assert.True(t, value)
	assert.True(t, cfg.SkipTLS)
	assert.True(t, cfg.CloudContext.SkipTLS)
}
