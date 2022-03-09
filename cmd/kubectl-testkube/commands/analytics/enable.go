package analytics

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewEnableAnalyticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Enable collecting of anonymous analytics",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.EnableAnalytics()
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("Analytics", "enabled")
		},
	}

	return cmd
}
