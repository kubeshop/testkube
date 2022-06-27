package telemetry

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewEnableTelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Enable collecting of anonymous telemetry data",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.EnableAnalytics()
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("Telemetry", "enabled")
		},
	}

	return cmd
}
