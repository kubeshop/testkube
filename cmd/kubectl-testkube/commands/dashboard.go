package commands

import (
	"fmt"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDashboardCmd is a method to create new dashboard command
func NewDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"d", "open-dashboard"},
		Short:   "Open Testkube Pro dashboard",
		Long:    `Open Testkube Pro dashboard`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.ContextType != config.ContextTypeCloud {
				ui.Warn("As of 1.17 the dashboard is no longer included with Testkube Core OSS - please refer to https://bit.ly/tk-dashboard for more info")
			} else {
				openCloudDashboard(cfg)
			}
		},
	}

	return cmd
}

func openCloudDashboard(cfg config.Data) {
	// open browser
	uri := fmt.Sprintf("%s/organization/%s/environment/%s", cfg.CloudContext.UiUri, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId)
	err := open.Run(uri)
	ui.PrintOnError("openning dashboard", err)
}
