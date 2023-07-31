package telemetry

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
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
				ui.PrintConfigError(err)
			} else {
				ui.PrintDisabled("Telemetry on CLI", "disabled")
			}

			client, _, err := common.GetClient(cmd)
			ui.WarnOnError("getting client", err)
			if err != nil {
				return
			}

			_, err = client.UpdateConfig(testkube.Config{EnableTelemetry: false})
			if err != nil {
				ui.PrintDisabled("Telemetry on API", "failed")
				ui.PrintConfigError(err)
			} else {
				ui.PrintDisabled("Telemetry on API", "disabled")
			}

			ui.NL()
		},
	}

	return cmd
}
