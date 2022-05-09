package analytics

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDisableAnalyticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "disable collecting of anonymous analytics",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.DisableAnalytics()

			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)

			ui.Success("Analytics", "disabled")
		},
	}

	return cmd
}
