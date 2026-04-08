package pro

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	docsUrl = "https://docs.testkube.io/testkube-pro/intro"
)

func NewConnectCmd() *cobra.Command {
	var opts = common.HelmOptions{}
	var skipExport bool
	var exportSince string

	cmd := &cobra.Command{
		Use:     "connect",
		Aliases: []string{"c"},
		Short:   "Testkube Pro connect ",
		Run: func(cmd *cobra.Command, args []string) {
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

			ui.H1("Connect your Pro environment:")
			ui.Paragraph("You can learn more about connecting your Testkube instance to Testkube Pro here:\n" + docsUrl)
			ui.H2("You can safely switch between connecting Pro and disconnecting without losing your data.")

			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)

			common.ProcessMasterFlags(cmd, &opts, &cfg)

			var clusterContext string
			var cliErr *common.CLIError
			if cfg.ContextType == config.ContextTypeKubeconfig {
				clusterContext, cliErr = common.GetCurrentKubernetesContext()
				common.HandleCLIError(cliErr)
			}

			// TODO: implement context info.
			ui.H1("Current status of your Testkube instance")
			ui.Properties([][]string{
				{"Context", apiContext},
				{"Kubectl context", clusterContext},
				{"Namespace", cfg.Namespace},
			})

			newStatus := [][]string{
				{"Testkube mode"},
				{"Context", contextDescription["cloud"]},
				{"Kubectl context", clusterContext},
				{"Namespace", cfg.Namespace},
				{ui.Separator, ""},
			}

			var (
				token        string
				refreshToken string
			)
			// if no agent is passed create new environment and get its token
			if opts.Master.AgentToken == "" && opts.Master.OrgId == "" && opts.Master.EnvId == "" {
				token, refreshToken, err = common.LoginUser(opts.Master.URIs.Auth, opts.Master.CustomAuth, opts.Master.CallbackPort)
				ui.ExitOnError("login", err)

				orgId, orgName, err := common.UiGetOrganizationId(opts.Master.URIs.Api, token)
				ui.ExitOnError("getting organization", err)

				envName, err := uiGetEnvName()
				ui.ExitOnError("getting environment name", err)

				envClient := cloudclient.NewEnvironmentsClient(opts.Master.URIs.Api, token, orgId)
				env, err := envClient.Create(cloudclient.Environment{Name: envName, Owner: orgId})
				ui.ExitOnError("creating environment", err)

				opts.Master.OrgId = orgId
				opts.Master.EnvId = env.Id
				opts.Master.AgentToken = env.AgentToken

				newStatus = append(
					newStatus,
					[][]string{
						{"Testkube will be connected to Pro org/env"},
						{"Organization Id", opts.Master.OrgId},
						{"Organization name", orgName},
						{"Environment Id", opts.Master.EnvId},
						{"Environment name", env.Name},
						{ui.Separator, ""},
					}...)
			}

			// validate if user created env - or was passed from flags
			if opts.Master.EnvId == "" {
				ui.Failf("You need pass valid environment id to connect to Pro")
			}
			if opts.Master.OrgId == "" {
				ui.Failf("You need pass valid organization id to connect to Pro")
			}

			// detect which database is currently deployed so we only scale down the active one
			dbType, cliErr := common.DetectDatabaseType(opts.Namespace)
			if cliErr != nil {
				common.HandleCLIError(cliErr)
			}
			cfg.CloudContext.DatabaseType = dbType

			// update summary
			newStatus = append(newStatus, []string{"Testkube support services not needed anymore"})
			newStatus = append(newStatus, []string{"MinIO     ", "Stopped and scaled down, (not deleted)"})
			switch dbType {
			case config.DatabaseTypeMongoDB:
				newStatus = append(newStatus, []string{"MongoDB   ", "Stopped and scaled down, (not deleted)"})
			case config.DatabaseTypePostgreSQL:
				newStatus = append(newStatus, []string{"PostgreSQL", "Stopped and scaled down, (not deleted)"})
			}

			ui.NL(2)

			ui.H1("Summary of your setup after connecting to Testkube Pro")
			ui.Properties(newStatus)

			ui.NL()
			ui.Warn("Remember: All your historical data and artifacts will be safe in case you want to rollback. OSS and Pro executions will be separated.")
			ui.NL()

			if ok := ui.Confirm("Proceed with connecting Testkube Pro?"); !ok {
				return
			}

			// Export execution data before switching to agent mode
			if !skipExport {
				ui.H2("Exporting execution data before connecting")
				exportPath, exportErr := client.ExportExecutions(".", exportSince)
				if exportErr != nil {
					if strings.Contains(exportErr.Error(), "413") {
						ui.Warn("Export archive exceeds the 100 MB size limit.")
						ui.Warn("Use the --since flag to limit the export to recent executions, e.g.: --since 2025-01-01")
					} else {
						ui.Warn(fmt.Sprintf("Warning: data export failed: %s", exportErr))
					}
					ui.Warn("Your data will remain in the existing database and can be exported later.")
					if ok := ui.Confirm("Continue connecting without export?"); !ok {
						return
					}
				} else {
					ui.Info(fmt.Sprintf("Export archive saved to: %s", exportPath))
				}
				ui.NL()
			}

			spinner := ui.NewSpinner("Connecting Testkube Pro")
			if cliErr = common.HelmUpgradeOrInstallTestkubeAgent(opts, cfg, true); cliErr != nil {
				spinner.Fail()
				common.HandleCLIError(cliErr)
			}

			spinner.Success()

			ui.NL()

			if opts.MinioReplicas == 0 {
				spinner = ui.NewSpinner("Scaling down MinIO")
				if _, scaleErr := common.KubectlScaleDeployment(opts.Namespace, "testkube-minio-testkube", opts.MinioReplicas); scaleErr != nil {
					spinner.Fail(fmt.Sprintf("Failed to scale down MinIO: %s", scaleErr))
				} else {
					spinner.Success()
				}
			}
			// scale down only the database that was originally deployed
			switch dbType {
			case config.DatabaseTypeMongoDB:
				if opts.MongoReplicas == 0 {
					spinner = ui.NewSpinner("Scaling down MongoDB")
					if _, scaleErr := common.KubectlScaleDeployment(opts.Namespace, "testkube-mongodb", opts.MongoReplicas); scaleErr != nil {
						spinner.Fail(fmt.Sprintf("Failed to scale down MongoDB: %s", scaleErr))
					} else {
						spinner.Success()
					}
				}
			case config.DatabaseTypePostgreSQL:
				if opts.PostgresReplicas == 0 {
					spinner = ui.NewSpinner("Scaling down PostgreSQL")
					if _, scaleErr := common.KubectlScaleStatefulSet(opts.Namespace, "testkube-postgresql-primary", opts.PostgresReplicas); scaleErr != nil {
						spinner.Fail(fmt.Sprintf("Failed to scale down PostgreSQL: %s", scaleErr))
					} else {
						spinner.Success()
					}
				}
			}

			ui.H2("Testkube Pro is connected to your Testkube instance, saving local configuration")

			ui.H2("Saving Testkube CLI Pro context")
			if token == "" && !common.IsUserLoggedIn(cfg, opts) {
				token, refreshToken, err = common.LoginUser(opts.Master.URIs.Auth, opts.Master.CustomAuth, opts.Master.CallbackPort)
				ui.ExitOnError("user login", err)
			}
			err = common.PopulateLoginDataToContext(opts.Master.OrgId, opts.Master.EnvId, token, refreshToken, "", opts, cfg)

			ui.ExitOnError("Setting Pro environment context", err)

			ui.NL(2)

			ui.ShellCommand("In case you want to roll back you can simply run the following command in your CLI:", "testkube pro disconnect")

			ui.Success("You can now login to Testkube Pro and validate your connection:")
			ui.NL()
			ui.Link(opts.Master.URIs.Ui + "/organization/" + opts.Master.OrgId + "/environment/" + opts.Master.EnvId + "/dashboard/tests")

			ui.NL(2)
		},
	}

	common.PopulateHelmFlags(cmd, &opts)
	common.PopulateMasterFlags(cmd, &opts, false)

	cmd.Flags().IntVar(&opts.MinioReplicas, "minio-replicas", 0, "MinIO replicas")
	cmd.Flags().IntVar(&opts.MongoReplicas, "mongo-replicas", 0, "MongoDB replicas")
	cmd.Flags().IntVar(&opts.PostgresReplicas, "postgres-replicas", 0, "PostgreSQL replicas")
	cmd.Flags().BoolVar(&skipExport, "skip-export", false, "Skip exporting execution data before connecting")
	cmd.Flags().StringVar(&exportSince, "since", "", "Export only executions created after this date (e.g. 2025-01-01 or 2025-01-01T00:00:00Z)")
	return cmd
}

var contextDescription = map[string]string{
	"":      "Unknown context, try updating your testkube cluster installation",
	"oss":   "Open Source Testkube",
	"cloud": "Testkube in Pro mode",
}

func uiGetEnvName() (string, error) {
	for i := 0; i < 3; i++ {
		if envName := ui.TextInput("Tell us the name of your environment"); envName != "" {
			return envName, nil
		}
		pterm.Error.Println("Environment name cannot be empty")
	}

	return "", fmt.Errorf("environment name cannot be empty")
}
