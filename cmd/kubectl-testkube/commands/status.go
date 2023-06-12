package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/telemetry"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "status [feature|resource]",
		Short:       "Show status of feature or resource",
		Annotations: map[string]string{cmdGroupAnnotation: cmdGroupCommands},
		Run: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Print(ui.IconRocket + "  Getting status on the testkube CLI")

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

			if cfg.OAuth2Data.Enabled {
				ui.PrintEnabled("OAuth", "enabled")
			} else {
				ui.PrintDisabled("Oauth", "disabled")
			}
			ui.NL()
		},
	}

	cmd.AddCommand(telemetry.NewStatusTelemetryCmd())
	cmd.AddCommand(oauth.NewStatusOAuthCmd())

	return cmd
}
