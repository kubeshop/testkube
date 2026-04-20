package pro

import (
	"fmt"

	"github.com/pterm/pterm"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDisconnectCmd() *cobra.Command {

	var (
		minioReplicas    int
		mongoReplicas    int
		postgresReplicas int
	)

	cmd := &cobra.Command{
		Use:     "disconnect",
		Aliases: []string{"d"},
		Short:   "Switch back to Testkube Core OSS mode, based on active .kube/config file",
		Run: func(cmd *cobra.Command, args []string) {

			ui.H1("Disconnecting your Pro environment:")
			ui.Paragraph("Rolling back to your clusters Testkube Core OSS installation")
			ui.Paragraph("If you need more details click into following link: " + docsUrl)
			ui.H2("You can safely switch between connecting Pro and disconnecting without losing your data.")

			cfg, err := config.Load()
			if err != nil {
				pterm.Error.Printfln("Failed to load config file: %s", err.Error())
				return
			}

			var clusterContext string
			var cliErr *common.CLIError
			if cfg.ContextType == config.ContextTypeKubeconfig {
				clusterContext, cliErr = common.GetCurrentKubernetesContext()
				common.HandleCLIError(cliErr)
			}

			ui.H1("Current status of your Testkube instance")

			summary := [][]string{
				{"Testkube mode"},
				{"Context", contextDescription["cloud"]},
				{"Kubectl context", clusterContext},
				{"Namespace", cfg.Namespace},
				{ui.Separator, ""},

				{"Testkube is connected to Pro organizations environment"},
				{"Organization Id", cfg.CloudContext.OrganizationId},
				{"Environment Id", cfg.CloudContext.EnvironmentId},
			}

			ui.Properties(summary)

			if ok := ui.Confirm("Shall we disconnect your Pro environment now?"); !ok {
				return
			}

			ui.NL(2)

			// uninstall the runner chart that was installed by "pro connect";
			// failures are non-fatal so disconnect can still restore OSS mode
			if cfg.CloudContext.AgentReleaseName != "" && cfg.CloudContext.AgentNamespace != "" {
				spinner := ui.NewSpinner("Uninstalling agent runner")
				if cliErr := common.HelmUninstall(cfg.CloudContext.AgentNamespace, cfg.CloudContext.AgentReleaseName); cliErr != nil {
					spinner.Fail(fmt.Sprintf("Failed to uninstall runner release %s (continuing with disconnect): %s", cfg.CloudContext.AgentReleaseName, cliErr))
				} else {
					spinner.Success()
				}
			}

			// Delete the agent record from the control plane that was created by "pro connect"
			if cfg.CloudContext.AgentName != "" && cfg.CloudContext.ApiUri != "" && cfg.CloudContext.ApiKey != "" && cfg.CloudContext.OrganizationId != "" {
				spinner := ui.NewSpinner("Deleting agent from control plane")
				if err := common.DeleteAgent(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, cfg.CloudContext.AgentName); err != nil {
					spinner.Fail(fmt.Sprintf("Failed to delete agent %q from control plane (continuing with disconnect): %s", cfg.CloudContext.AgentName, err))
				} else {
					spinner.Success()
				}
			}

			ns := cfg.Namespace

			// Scale up the OSS API server that was scaled down by pro connect
			spinner := ui.NewSpinner("Scaling up testkube-api-server")
			if _, scaleErr := common.KubectlScaleDeployment(ns, "testkube-api-server", 1); scaleErr != nil {
				spinner.Fail(fmt.Sprintf("Failed to scale up testkube-api-server: %s", scaleErr))
			} else {
				spinner.Success()
			}

			// Restore support services that were scaled down by pro connect
			if minioReplicas > 0 {
				spinner = ui.NewSpinner("Scaling up MinIO")
				if _, scaleErr := common.KubectlScaleDeployment(ns, "testkube-minio-testkube", minioReplicas); scaleErr != nil {
					spinner.Fail(fmt.Sprintf("Failed to scale up MinIO: %s", scaleErr))
				} else {
					spinner.Success()
				}
			}
			spinner = ui.NewSpinner("Scaling up NATS")
			if _, scaleErr := common.KubectlScaleStatefulSet(ns, "testkube-nats", 1); scaleErr != nil {
				spinner.Fail(fmt.Sprintf("Failed to scale up NATS: %s", scaleErr))
			} else {
				spinner.Success()
			}
			dbType := cfg.CloudContext.DatabaseType
			switch dbType {
			case config.DatabaseTypeMongoDB:
				if mongoReplicas > 0 {
					spinner = ui.NewSpinner("Scaling up MongoDB")
					if _, scaleErr := common.KubectlScaleDeployment(ns, "testkube-mongodb", mongoReplicas); scaleErr != nil {
						spinner.Fail(fmt.Sprintf("Failed to scale up MongoDB: %s", scaleErr))
					} else {
						spinner.Success()
					}
				}
			case config.DatabaseTypePostgreSQL:
				if postgresReplicas > 0 {
					spinner = ui.NewSpinner("Scaling up PostgreSQL")
					if _, scaleErr := common.KubectlScaleStatefulSet(ns, "testkube-postgresql", postgresReplicas); scaleErr != nil {
						spinner.Fail(fmt.Sprintf("Failed to scale up PostgreSQL: %s", scaleErr))
					} else {
						spinner.Success()
					}
				}
			default:
				// no database type recorded – fall back to attempting both so that clusters
				// connected before this feature was introduced are handled gracefully;
				// errors are silently ignored because only one DB is actually deployed
				if mongoReplicas > 0 {
					if _, scaleErr := common.KubectlScaleDeployment(ns, "testkube-mongodb", mongoReplicas); scaleErr == nil {
						ui.Success("Scaled up MongoDB")
					}
				}
				if postgresReplicas > 0 {
					if _, scaleErr := common.KubectlScaleStatefulSet(ns, "testkube-postgresql", postgresReplicas); scaleErr == nil {
						ui.Success("Scaled up PostgreSQL")
					}
				}
			}

			spinner = ui.NewSpinner("Resetting Testkube config.json")
			cfg.ContextType = config.ContextTypeKubeconfig
			cfg.CloudContext = config.CloudContext{}
			if err = config.Save(cfg); err != nil {
				spinner.Fail(fmt.Sprintf("Error updating local Testkube config file: %s", err))
				ui.Warn("Please manually remove the fields contextType and cloudContext from your config file.")
			} else {
				spinner.Success()
			}

			ui.NL()
			ui.Success("Disconnect finished successfully")
			ui.NL()
		},
	}

	cmd.Flags().IntVar(&minioReplicas, "minio-replicas", 1, "MinIO replicas to restore on disconnect")
	cmd.Flags().IntVar(&mongoReplicas, "mongo-replicas", 1, "MongoDB replicas to restore on disconnect")
	cmd.Flags().IntVar(&postgresReplicas, "postgres-replicas", 1, "PostgreSQL replicas to restore on disconnect")
	return cmd
}
