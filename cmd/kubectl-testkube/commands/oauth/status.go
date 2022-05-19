package oauth

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

// NewStatusOAuthCmd is oauth status command
func NewStatusOAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "Get oauth status",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.OAuth2Data.Enabled {
				ui.Success("OAuth", "enabled")
			} else {
				ui.Success("OAuth", "disabled")
			}
		},
	}

	return cmd
}
