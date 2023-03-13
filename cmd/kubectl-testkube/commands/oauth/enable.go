package oauth

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewEnableOAuthCmd is oauth enable command
func NewEnableOAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "enable oauth authentication for direct api",
		Run: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Print(ui.IconRocket + "  Enabling OAuth authentication for direct api")
			cfg, err := config.Load()
			if err == nil {
				cfg.EnableOAuth()
				err = config.Save(cfg)
			}
			if err != nil {
				ui.PrintDisabled("OAuth", "failed")
				ui.PrintConfigError(err)
			} else {
				ui.PrintEnabled("OAuth", "enabled")
			}
			ui.NL()
		},
	}

	return cmd
}
