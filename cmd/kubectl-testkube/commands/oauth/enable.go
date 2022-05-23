package oauth

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

// NewEnableOAuthCmd is oauth enable command
func NewEnableOAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "enable oauth authentication for direct api",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.EnableOAuth()

			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("OAuth", "enabled")
		},
	}

	return cmd
}
