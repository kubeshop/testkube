package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewConfigureDashboardNameCmd is dashboard name config command
func NewConfigureDashboardNameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard-name <value>",
		Short: "Set dashboard name for testkube client",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please pass valid dashboard name value")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.DashboardName = args[0]
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New dashboard name set to", cfg.DashboardName)
		},
	}

	return cmd
}
