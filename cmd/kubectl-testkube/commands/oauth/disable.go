package oauth

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

// NewDisableOAuthCmd is oauth disable command
func NewDisableOAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "disable oauth authentication for direct api",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.DisableOauth()

			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("OAuth", "disabled")
		},
	}

	return cmd
}
