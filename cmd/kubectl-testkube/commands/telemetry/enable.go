package telemetry

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewEnableTelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Enable collecting of anonymous telemetry data",
		Run: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Print(ui.IconRocket + "  Enabling telemetry on the testkube CLI")

			cfg, err := config.Load()
			if err == nil {
				cfg.EnableAnalytics()
				err = config.Save(cfg)
			}
			if err != nil {
				ui.PrintDisabled("Telemetry on CLI", "failed")
				ui.PrintConfigError(err)
			} else {
				ui.PrintEnabled("Telemetry on CLI", "enabled")
			}

			client, _, err := common.GetClient(cmd)
			ui.WarnOnError("getting client", err)
			if err != nil {
				return
			}

			_, err = client.UpdateConfig(testkube.Config{EnableTelemetry: true})
			if err != nil {
				ui.PrintDisabled("Telemetry on API", "failed")
				ui.PrintConfigError(err)
			} else {
				ui.PrintEnabled("Telemetry on API", "enabled")
			}

			ui.NL()
		},
	}

	return cmd
}
