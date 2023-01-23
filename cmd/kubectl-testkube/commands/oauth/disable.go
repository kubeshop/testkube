package oauth

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDisableOAuthCmd is oauth disable command
func NewDisableOAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "disable oauth authentication for direct api",
		Run: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Print(ui.IconRocket + "  Disabling OAuth authentication for direct api")
			cfg, err := config.Load()

			if err == nil {
				cfg.DisableOauth()
				err = config.Save(cfg)
			}
			if err != nil {
				ui.PrintDisabled("OAuth", "failed")
				ui.PrintConfigError(err)
			} else {
				ui.PrintDisabled("OAuth", "disabled")
			}
			ui.NL()
		},
	}

	return cmd
}
