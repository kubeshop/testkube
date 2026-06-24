package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
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

			printContextStatus(cmd, cfg)

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

	cmd.AddCommand(telemetry.NewStatusTelemetryCmd())

	return cmd
}

// printContextStatus reports whether the CLI is talking to the Testkube
// Control Plane (cloud context) or a local/standalone agent (kubeconfig
// context), along with the connection details relevant to that context.
func printContextStatus(cmd *cobra.Command, cfg config.Data) {
	if cfg.ContextType == config.ContextTypeCloud {
		ui.PrintEnabled("Context", "connected to Testkube Control Plane")

		orgName := cfg.CloudContext.OrganizationName
		if orgName == "" {
			orgName = cfg.CloudContext.OrganizationId
		}
		envName := cfg.CloudContext.EnvironmentName
		if envName == "" {
			envName = cfg.CloudContext.EnvironmentId
		}

		contextData := map[string]string{
			"Organization": orgName,
			"Environment ": envName,
			"API URI     ": cfg.CloudContext.ApiUri,
			"Namespace   ": cfg.Namespace,
		}
		ui.InfoGrid(contextData)
	} else {
		ui.PrintEnabled("Context", "connected to a local standalone agent")

		namespace := cfg.Namespace
		if flag := cmd.Flag("namespace"); flag != nil && flag.Value.String() != "" {
			namespace = flag.Value.String()
		}
		apiURI := cfg.APIURI
		if flag := cmd.Flag("api-uri"); flag != nil && flag.Value.String() != "" {
			apiURI = flag.Value.String()
		}

		ui.InfoGrid(map[string]string{
			"Namespace": namespace,
			"API URI  ": apiURI,
		})
	}

	ui.NL()
}
