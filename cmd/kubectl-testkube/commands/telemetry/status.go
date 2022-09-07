package telemetry

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewStatusTelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Get telemetry status",
		Run: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Print(ui.IconRocket + "  Getting telemetry status on the testkube CLI")

			cfg, err := config.Load()
			ui.ExitOnError("   Loading config file failed", err)
			if cfg.TelemetryEnabled {
				ui.PrintEnabled("Telemetry on CLI", "enabled")
			} else {
				ui.PrintDisabled("Telemetry on CLI", "disabled")
			}
			ui.NL()
		},
	}

	return cmd
}
