package analytics

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewStatusAnalyticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get analytics status",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			if config.Config.AnalyticsEnabled {
				ui.Success("Analytics", "enabled")
			} else {
				ui.Success("Analytics", "disabled")
			}
		},
	}

	return cmd
}
