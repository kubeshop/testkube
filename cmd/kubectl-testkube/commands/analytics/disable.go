package analytics

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDisableAnalyticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disable",
		Aliases: []string{"on", "d", "n"},
		Short:   "disable collecting of anonymous analytics",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			config.Config.DisableAnalytics()
			err := config.Config.Save(config.Config.Data)
			ui.ExitOnError("saving config file", err)
			ui.Success("Analytics", "disabled")
		},
	}

	return cmd
}
