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
			ui.NL()
			ui.Print(ui.IconRocket + "  Disabling telemetry on the testkube CLI")

			cfg, err := config.Load()
			if err == nil {
				cfg.DisableAnalytics()
				err = config.Save(cfg)
			}
			if err != nil {
				ui.PrintDisabled("Telemetry on CLI", "failed")
				ui.PrintOnError("    Can't access config file", err)
				ui.Info(ui.IconSuggestion+"  Suggestion:", "Do you have enough rights to handle the config file?")
				ui.Info(ui.IconDocumentation+"  Documentation:", "https://kubeshop.github.io/testkube/")
			} else {
				ui.PrintDisabled("Telemetry on CLI", "disabled")
			}
			ui.NL()
		},
	}

	return cmd
}
