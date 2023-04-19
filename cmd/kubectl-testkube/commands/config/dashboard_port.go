package config

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewConfigureDashboardPortCmd is dashboard port config command
func NewConfigureDashboardPortCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard-port <value>",
		Short: "Set dashboard port for testkube client",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please pass valid dashboard port value")
			}

			if _, err := strconv.Atoi(args[0]); err != nil {
				return fmt.Errorf("please pass integer dashboard port value: %w", err)
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			port, err := strconv.Atoi(args[0])
			ui.ExitOnError("converting port value", err)

			cfg.DashboardPort = port
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New dashboard port set to", strconv.Itoa(cfg.DashboardPort))
		},
	}

	return cmd
}
