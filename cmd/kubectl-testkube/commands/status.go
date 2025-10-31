package commands

import (
	"fmt"

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

			// Display current context information
			ui.NL()
			ui.Info("Context:", string(cfg.ContextType))
			
			if cfg.ContextType == config.ContextTypeCloud {
				orgName := cfg.CloudContext.OrganizationName
				if orgName == "" {
					orgName = cfg.CloudContext.OrganizationId
				}
				envName := cfg.CloudContext.EnvironmentName
				if envName == "" {
					envName = cfg.CloudContext.EnvironmentId
				}
				
				ui.Info("  Organization:", orgName)
				ui.Info("  Environment:", envName)
				ui.Info("  API URI:", cfg.CloudContext.ApiUri)
			} else {
				ui.Info("  Namespace:", cfg.Namespace)
			}

			// Try to connect to the environment
			ui.NL()
			client, _, err := common.GetClient(cmd)
			if err != nil {
				ui.PrintDisabled("Environment Connectivity", fmt.Sprintf("failed (%s)", err.Error()))
				ui.Warn("Unable to connect to Testkube environment")
				ui.NL()
				return
			}

			// Validate connectivity by getting server info
			serverInfo, err := client.GetServerInfo()
			if err != nil {
				ui.PrintDisabled("Environment Connectivity", fmt.Sprintf("failed (%s)", err.Error()))
				ui.Warn("Unable to reach Testkube environment")
				ui.NL()
				return
			}

			ui.PrintEnabled("Environment Connectivity", "reachable")
			ui.Info("  Server Version:", serverInfo.Version)
			if serverInfo.Context != "" {
				ui.Info("  Server Context:", serverInfo.Context)
			}
			if serverInfo.Namespace != "" {
				ui.Info("  Server Namespace:", serverInfo.Namespace)
			}

			// Get API configuration
			ui.NL()
			config, err := client.GetConfig()
			if err != nil {
				ui.Warn("Unable to get API config:", err.Error())
			} else {
				if config.EnableTelemetry {
					ui.PrintEnabled("Telemetry on API", "enabled")
				} else {
					ui.PrintDisabled("Telemetry on API", "disabled")
				}
			}

			if cfg.TelemetryEnabled {
				ui.PrintEnabled("Telemetry on CLI", "enabled")
			} else {
				ui.PrintDisabled("Telemetry on CLI", "disabled")
			}

			ui.NL()
		},
	}

	cmd.AddCommand(telemetry.NewStatusTelemetryCmd())

	return cmd
}
