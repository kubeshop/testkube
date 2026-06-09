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
			// genuine on->off transition - and only while it is still enabled.
			wasEnabled := err == nil && cfg.TelemetryEnabled

			client, _, clientErr := common.GetClient(cmd)
			ui.WarnOnError("getting client", clientErr)

			apiDisabled := false
			if clientErr == nil {
				if _, apiErr := client.UpdateConfig(testkube.Config{EnableTelemetry: false}); apiErr != nil {
					ui.PrintDisabled("Telemetry on API", "failed")
					ui.PrintConfigError(apiErr)
				} else {
					ui.PrintDisabled("Telemetry on API", "disabled")
					apiDisabled = true
				}
			}

			// Emit the final opt-out event while telemetry is still enabled
			// locally, but only once the API opt-out has persisted. If the API
			// update failed, the root post-run sync would re-enable the CLI
			// config, so sending here would record a false opt-out.
			if wasEnabled && apiDisabled {
				if _, sendErr := telemetry.SendTelemetryOptOutEvent(cmd, common.Version); sendErr != nil {
					ui.Debug("sending telemetry opt-out event failed", sendErr.Error())
				}
			}

			// Persist the local opt-out last, after the final event has been sent.
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

			ui.NL()
		},
	}

	return cmd
}
