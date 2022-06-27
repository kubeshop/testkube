package telemetry

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDisableTelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "disable collecting of anonymous telemetry data",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.DisableAnalytics()

			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)

			ui.Success("Telemetry", "disabled")
		},
	}

	return cmd
}
