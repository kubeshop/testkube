package telemetry

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewStatusTelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Get telemetry status",
		Run: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Print(ui.IconRocket + "  Getting telemetry status on the testkube CLI and API")

			cfg, err := config.Load()
			ui.ExitOnError("   Loading config file failed", err)
			if cfg.TelemetryEnabled {
				ui.PrintEnabled("Telemetry on CLI", "enabled")
			} else {
				ui.PrintDisabled("Telemetry on CLI", "disabled")
			}

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			config, err := client.GetConfig()
			ui.ExitOnError("   Getting API config failed", err)
			if config.EnableTelemetry {
				ui.PrintEnabled("Telemetry on API", "enabled")
			} else {
				ui.PrintDisabled("Telemetry on API", "disabled")
			}

			ui.NL()
		},
	}

	return cmd
}
