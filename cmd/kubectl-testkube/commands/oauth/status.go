package oauth

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewStatusOAuthCmd is oauth status command
func NewStatusOAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "Get oauth status",
		Run: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Print(ui.IconRocket + "  Getting OAuth status")

			cfg, err := config.Load()
			ui.ExitOnError("   Loading config file failed", err)
			if cfg.OAuth2Data.Enabled {
				ui.PrintEnabled("OAuth", "enabled")
			} else {
				ui.PrintDisabled("OAuth", "disabled")
			}
			ui.NL()
		},
	}

	return cmd
}
