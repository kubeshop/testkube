package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstallLocal_NeverReachesTheRealClusterPurge(t *testing.T) {
	root := &cobra.Command{Use: "testkube"}
	root.AddCommand(NewPurgeCmd())
	tests := map[string]struct {
		args []string
		want string
	}{
		"uninstall local is ours":      {[]string{"uninstall", "local"}, "local"},
		"purge local is ours":          {[]string{"purge", "local"}, "local"},
		"plain uninstall still purges": {[]string{"uninstall"}, "purge"},
		"other words still purge":      {[]string{"purge", "something"}, "purge"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.want, cmd.Name())
		})
	}
}
