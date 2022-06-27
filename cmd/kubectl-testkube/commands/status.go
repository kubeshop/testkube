package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/telemetry"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [feature|resource]",
		Short: "Show status of feature or resource",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.TelemetryEnabled {
				ui.Success("Telemetry", "enabled")
			} else {
				ui.Success("Telemetry", "disabled")
			}

			if cfg.OAuth2Data.Enabled {
				ui.Success("OAuth", "enabled")
			} else {
				ui.Success("OAuth", "disabled")
			}
		},
	}

	cmd.AddCommand(telemetry.NewStatusTelemetryCmd())
	cmd.AddCommand(oauth.NewStatusOAuthCmd())

	return cmd
}
