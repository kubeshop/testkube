package pro

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

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
	var setOptions, argOptions map[string]string
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

			// validate required flags
			if opts.Master.AgentToken == "" {
				ui.Failf("You need to pass a valid agent token to connect to Pro (--agent-token)")
			}
			if opts.Master.OrgId == "" {
				ui.Failf("You need to pass a valid organization id to connect to Pro (--org-id)")
			}
			if opts.Master.EnvId == "" {
				ui.Failf("You need to pass a valid environment id to connect to Pro (--env-id)")
			}

			newStatus := [][]string{
				{"Testkube mode"},
				{"Context", contextDescription["cloud"]},
				{"Kubectl context", clusterContext},
				{"Namespace", cfg.Namespace},
				{ui.Separator, ""},
				{"Organization Id", opts.Master.OrgId},
				{"Environment Id", opts.Master.EnvId},
			}

			// detect which database is currently deployed so we only scale down the active one
			dbType, cliErr := common.DetectDatabaseType(opts.Namespace)
			if cliErr != nil {
				common.HandleCLIError(cliErr)
			}
			cfg.CloudContext.DatabaseType = dbType

			// update summary
			newStatus = append(newStatus, []string{ui.Separator, ""})
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
			var exportPath string
			if !skipExport {
				ui.H2("Exporting execution data before connecting")
				var exportErr error
				exportPath, exportErr = client.ExportExecutions(".", exportSince)
				if exportErr != nil {
					var httpErr *cloudclient.HTTPError
					is413 := errors.As(exportErr, &httpErr) && httpErr.StatusCode == http.StatusRequestEntityTooLarge
					if !is413 {
						is413 = strings.Contains(exportErr.Error(), strconv.Itoa(http.StatusRequestEntityTooLarge))
					}
					if is413 {
						ui.Warn("Export archive exceeds the server size limit.")
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
			opts.SetOptions = setOptions
			opts.ArgOptions = argOptions
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

			ui.H2("Saving Testkube CLI Pro context")
			cfg.ContextType = config.ContextTypeCloud
			err = common.PopulateAgentDataToContext(opts, cfg)
			ui.ExitOnError("Setting Pro environment context", err)

			// Upload the previously exported archive to the control plane
			if exportPath != "" {
				// Resolve effective credential: use agent token from flags,
				// otherwise fall back to the value already stored in the config.
				effectiveToken := opts.Master.AgentToken
				if effectiveToken == "" {
					effectiveToken = cfg.CloudContext.AgentKey
				}
				spinner = ui.NewSpinner("Importing execution data to the control plane")
				importClient := cloudclient.NewImportClient(opts.Master.URIs.Api, effectiveToken, opts.Master.OrgId, opts.Master.EnvId)
				importErr := importClient.Import(cmd.Context(), exportPath)
				if importErr != nil {
					var httpErr *cloudclient.HTTPError
					if errors.As(importErr, &httpErr) && httpErr.StatusCode == http.StatusRequestEntityTooLarge {
						spinner.Fail("Import archive exceeds the server size limit. Use the --since flag to limit the export to recent executions, e.g.: --since 2025-01-01")
					} else {
						spinner.Fail(fmt.Sprintf("Failed to import execution data: %s", importErr))
					}
					ui.Warn("The exported archive is still available at: " + exportPath)
				} else {
					spinner.Success()
					ui.Info("Archive processing is done asynchronously and can take up to a few minutes depending on the number of executions.")
					// Clean up the exported archive after successful import
					if removeErr := os.Remove(exportPath); removeErr != nil {
						ui.Warn(fmt.Sprintf("Warning: could not remove export file %s: %s", exportPath, removeErr))
					}
				}
			}

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
	cmd.Flags().BoolVar(&opts.MultiNamespace, "multi-namespace", false, "multi namespace mode")
	cmd.Flags().BoolVar(&opts.NoCRDs, "no-crds", false, "Skip installing CRDs, useful when you have them already installed in the cluster")
	cmd.Flags().StringToStringVarP(&setOptions, "helm-set", "", nil, "helm set option in form of key=value")
	cmd.Flags().StringToStringVarP(&argOptions, "helm-arg", "", nil, "helm arg option in form of key=value")
	cmd.Flags().BoolVar(&skipExport, "skip-export", false, "Skip exporting execution data before connecting")
	cmd.Flags().StringVar(&exportSince, "since", "", "Export only executions created after this date (e.g. 2025-01-01 or 2025-01-01T00:00:00Z)")
	return cmd
}

var contextDescription = map[string]string{
	"":      "Unknown context, try updating your testkube cluster installation",
	"oss":   "Open Source Testkube",
	"cloud": "Testkube in Pro mode",
}
