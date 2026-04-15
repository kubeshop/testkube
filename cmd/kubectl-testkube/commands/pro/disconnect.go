package pro

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDisconnectCmd() *cobra.Command {

	var opts common.HelmOptions

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

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			info, err := client.GetServerInfo()
			firstInstall := err != nil && strings.Contains(err.Error(), "not found")
			if err != nil && !firstInstall {
				ui.Failf("Can't get Testkube cluster information: %s", err.Error())
			}
			var apiContext string
			if actx, ok := contextDescription[info.Context]; ok {
				apiContext = actx
			}
			var clusterContext string
			var cliErr *common.CLIError
			if cfg.ContextType == config.ContextTypeKubeconfig {
				clusterContext, cliErr = common.GetCurrentKubernetesContext()
				common.HandleCLIError(cliErr)
			}

			// TODO: implement context info
			ui.H1("Current status of your Testkube instance")

			summary := [][]string{
				{"Testkube mode"},
				{"Context", apiContext},
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

			spinner := ui.NewSpinner("Disconnecting from Testkube Pro")

			// ensure only the originally-active database is re-enabled in the Helm release;
			// the user-supplied --no-mongo/--no-postgres flags take precedence if set explicitly
			dbType := cfg.CloudContext.DatabaseType
			switch dbType {
			case config.DatabaseTypeMongoDB:
				// original DB was Mongo – keep Postgres disabled unless the user explicitly enabled it
				if !cmd.Flags().Changed("no-postgres") {
					opts.NoPostgres = true
				}
			case config.DatabaseTypePostgreSQL:
				// original DB was Postgres – keep Mongo disabled and re-enable Postgres
				if !cmd.Flags().Changed("no-mongo") {
					opts.NoMongo = true
				}
				if !cmd.Flags().Changed("no-postgres") {
					opts.NoPostgres = false
				}
			default:
				// DatabaseType was never recorded (cluster connected before this feature).
				// Old clusters only ever had MongoDB, so keep PostgreSQL disabled.
				if !cmd.Flags().Changed("no-postgres") {
					opts.NoPostgres = true
				}
			}

			if cliErr := common.HelmUpgradeOrInstallTestkube(opts); cliErr != nil {
				spinner.Fail()
				common.HandleCLIError(cliErr)
			}

			spinner.Success()

			// restore the database that was originally deployed before connecting to Pro
			if opts.MinioReplicas > 0 {
				spinner = ui.NewSpinner("Scaling up MinIO")
				if _, scaleErr := common.KubectlScaleDeployment(opts.Namespace, "testkube-minio-testkube", opts.MinioReplicas); scaleErr != nil {
					spinner.Fail(fmt.Sprintf("Failed to scale up MinIO: %s", scaleErr))
				} else {
					spinner.Success()
				}
			}
			switch dbType {
			case config.DatabaseTypeMongoDB:
				if opts.MongoReplicas > 0 {
					spinner = ui.NewSpinner("Scaling up MongoDB")
					if _, scaleErr := common.KubectlScaleDeployment(opts.Namespace, "testkube-mongodb", opts.MongoReplicas); scaleErr != nil {
						spinner.Fail(fmt.Sprintf("Failed to scale up MongoDB: %s", scaleErr))
					} else {
						spinner.Success()
					}
				}
			case config.DatabaseTypePostgreSQL:
				if opts.PostgresReplicas > 0 {
					spinner = ui.NewSpinner("Scaling up PostgreSQL")
					if _, scaleErr := common.KubectlScaleStatefulSet(opts.Namespace, "testkube-postgresql", opts.PostgresReplicas); scaleErr != nil {
						spinner.Fail(fmt.Sprintf("Failed to scale up PostgreSQL: %s", scaleErr))
					} else {
						spinner.Success()
					}
				}
			default:
				// no database type recorded – fall back to attempting both so that clusters
				// connected before this feature was introduced are handled gracefully;
				// errors are silently ignored because only one DB is actually deployed
				if opts.MongoReplicas > 0 {
					if _, scaleErr := common.KubectlScaleDeployment(opts.Namespace, "testkube-mongodb", opts.MongoReplicas); scaleErr == nil {
						ui.Success("Scaled up MongoDB")
					}
				}
				if opts.PostgresReplicas > 0 {
					if _, scaleErr := common.KubectlScaleStatefulSet(opts.Namespace, "testkube-postgresql", opts.PostgresReplicas); scaleErr == nil {
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

	// populate options
	common.PopulateHelmFlags(cmd, &opts)
	cmd.Flags().IntVar(&opts.MinioReplicas, "minio-replicas", 1, "MinIO replicas")
	cmd.Flags().IntVar(&opts.MongoReplicas, "mongo-replicas", 1, "MongoDB replicas")
	cmd.Flags().IntVar(&opts.PostgresReplicas, "postgres-replicas", 1, "PostgreSQL replicas")
	return cmd
}
