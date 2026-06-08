package telemetry

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/telemetry"
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
			// Remember whether telemetry was actually on, so we only record a
			// genuine on->off transition once we know the opt-out persisted.
			wasEnabled := err == nil && cfg.TelemetryEnabled
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
				// Only record the opt-out now that it has persisted on both the
				// CLI and the API. If the API update had failed, the root
				// post-run sync would re-enable the CLI config, leaving the user
				// opted in - so sending earlier would produce a false opt-out.
				if wasEnabled {
					if _, sendErr := telemetry.SendTelemetryOptOutEvent(cmd, common.Version); sendErr != nil {
						ui.Debug("sending telemetry opt-out event failed", sendErr.Error())
					}
				}
			}

			ui.NL()
		},
	}

	return cmd
}
