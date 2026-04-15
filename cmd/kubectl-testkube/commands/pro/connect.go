package pro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/agents"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	docsUrl = "https://docs.testkube.io/testkube-pro/intro"
)

func NewConnectCmd() *cobra.Command {
	var (
		// Cloud/master flags (resolved via PopulateMasterFlags + ProcessMasterFlags)
		masterOpts common.HelmOptions

		// Export/import flags
		skipExport  bool
		exportSince string
	)

	cmd := &cobra.Command{
		Use:     "connect [agent-name]",
		Aliases: []string{"c"},
		Args:    cobra.MaximumNArgs(1),
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

			// detect which database is currently deployed so we only scale down the active one
			dbType, cliErr := common.DetectDatabaseType(cfg.Namespace)
			if cliErr != nil {
				common.HandleCLIError(cliErr)
			}
			cfg.CloudContext.DatabaseType = dbType

			// update summary
			newStatus := [][]string{
				{"Testkube mode"},
				{"Context", contextDescription["cloud"]},
				{"Kubectl context", clusterContext},
				{"Namespace", cfg.Namespace},
				{ui.Separator, ""},
				{"Testkube support services not needed anymore"},
				{"MinIO     ", "Stopped and scaled down, (not deleted)"},
			}
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

			// Populate cloud context from the master flags (--org-id, --env-id,
			// --root-domain, --agent-token, etc.) so that UiInstallAgent and
			// control plane API calls use the correct URIs and credentials.
			common.ProcessMasterFlags(cmd, &masterOpts, &cfg)
			err = common.PopulateAgentDataToContext(masterOpts, cfg)
			ui.ExitOnError("populating cloud context from flags", err)

			// Reload after PopulateAgentDataToContext may have saved
			cfg, err = config.Load()
			ui.ExitOnError("reloading config", err)

			// Switch CLI context to cloud mode
			cfg.ContextType = config.ContextTypeCloud
			err = config.Save(cfg)
			ui.ExitOnError("saving cloud context configuration", err)

			// Install agent using same mechanism as "install agent" command
			agentName := "default-oss"
			if len(args) > 0 && args[0] != "" {
				agentName = args[0]
			}

			// Set pro connect defaults: enable all capabilities, auto-create, global
			for _, flag := range []string{"runner", "listener", "gitops", "webhooks", "create", "global"} {
				if !cmd.Flags().Changed(flag) {
					_ = cmd.Flags().Set(flag, "true")
				}
			}
			// Default the namespace to the current OSS installation namespace
			if !cmd.Flags().Changed("namespace") {
				_ = cmd.Flags().Set("namespace", cfg.Namespace)
			}

			agents.UiInstallAgent(cmd, agentName, []string{"testkube.io/source=oss"}, map[string]interface{}{
				// Disable CRD installation in the runner chart — the OSS chart already
				// has the CRDs installed via the testkube-operator subchart. This prevents
				// duplicate ownership and ensures disconnect (helm uninstall) does not
				// remove CRDs still needed by the OSS chart.
				"gitops.installCRD": false,
			})

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			// Check if dry-run mode — skip post-install steps
			if dryRun {
				return
			}

			// Scale down old support services in the original namespace
			origNs := cfg.Namespace
			spinner := ui.NewSpinner("Scaling down MinIO")
			if _, scaleErr := common.KubectlScaleDeployment(origNs, "testkube-minio-testkube", 0); scaleErr != nil {
				spinner.Fail(fmt.Sprintf("Failed to scale down MinIO: %s", scaleErr))
			} else {
				spinner.Success()
			}
			// scale down only the database that was originally deployed
			switch dbType {
			case config.DatabaseTypeMongoDB:
				spinner = ui.NewSpinner("Scaling down MongoDB")
				if _, scaleErr := common.KubectlScaleDeployment(origNs, "testkube-mongodb", 0); scaleErr != nil {
					spinner.Fail(fmt.Sprintf("Failed to scale down MongoDB: %s", scaleErr))
				} else {
					spinner.Success()
				}
			case config.DatabaseTypePostgreSQL:
				spinner = ui.NewSpinner("Scaling down PostgreSQL")
				if _, scaleErr := common.KubectlScaleStatefulSet(origNs, "testkube-postgresql-primary", 0); scaleErr != nil {
					spinner.Fail(fmt.Sprintf("Failed to scale down PostgreSQL: %s", scaleErr))
				} else {
					spinner.Success()
				}
			}

			// Save context
			ui.H2("Saving Testkube CLI Pro context")
			cfg, err = config.Load()
			ui.ExitOnError("loading config", err)
			cfg.CloudContext.DatabaseType = dbType
			cfg.ContextType = config.ContextTypeCloud
			// detect the runner release installed by UiInstallAgent so disconnect can uninstall it
			if relName, relNs := findRunnerRelease(); relName != "" {
				cfg.CloudContext.AgentReleaseName = relName
				cfg.CloudContext.AgentNamespace = relNs
			}
			err = config.Save(cfg)
			ui.ExitOnError("Saving Pro context", err)

			// Upload the previously exported archive to the control plane
			if exportPath != "" {
				effectiveToken := cfg.CloudContext.ApiKey
				if effectiveToken == "" {
					effectiveToken = cfg.CloudContext.AgentKey
				}
				if cfg.CloudContext.OrganizationId != "" && cfg.CloudContext.EnvironmentId != "" && effectiveToken != "" {
					spinner = ui.NewSpinner("Importing execution data to the control plane")
					importClient := cloudclient.NewImportClient(
						cfg.CloudContext.ApiUri,
						effectiveToken,
						cfg.CloudContext.OrganizationId,
						cfg.CloudContext.EnvironmentId,
					)
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
				} else {
					ui.Warn("Skipping import: organization ID, environment ID, or API token not available in config.")
					ui.Warn("The exported archive is available at: " + exportPath)
				}
			}

			ui.NL(2)

			ui.ShellCommand("In case you want to roll back you can simply run the following command in your CLI:", "testkube pro disconnect")

			ui.Success("You can now login to Testkube Pro and validate your connection")

			ui.NL(2)
		},
	}

	common.PopulateRunnerFlags(cmd, false)

	// Export/import flags
	cmd.Flags().BoolVar(&skipExport, "skip-export", false, "Skip exporting execution data before connecting")
	cmd.Flags().StringVar(&exportSince, "since", "", "Export only executions created after this date (e.g. 2025-01-01 or 2025-01-01T00:00:00Z)")

	// Cloud/master flags (--org-id, --env-id, --root-domain, --agent-token, etc.)
	common.PopulateMasterFlags(cmd, &masterOpts, false)

	return cmd
}

// helmRelease represents a single entry from `helm list --output json`.
type helmRelease struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Chart     string `json:"chart"`
}

// findRunnerRelease uses `helm list` to find a testkube-runner release installed by pro connect.
// Returns empty strings if helm is not available or no runner release is found.
func findRunnerRelease() (releaseName, namespace string) {
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		ui.Debug("findRunnerRelease: helm not found in PATH")
		return "", ""
	}
	out, execErr := exec.Command(helmPath, "list", "--all-namespaces", "--output", "json").CombinedOutput()
	if execErr != nil {
		ui.Debug(fmt.Sprintf("findRunnerRelease: helm list failed: %s", execErr))
		return "", ""
	}
	var releases []helmRelease
	if jsonErr := json.Unmarshal(out, &releases); jsonErr != nil {
		ui.Debug(fmt.Sprintf("findRunnerRelease: failed to parse helm list output: %s", jsonErr))
		return "", ""
	}
	for _, r := range releases {
		if strings.HasPrefix(r.Chart, "testkube-runner-") {
			return r.Name, r.Namespace
		}
	}
	ui.Debug("findRunnerRelease: no testkube-runner release found")
	return "", ""
}

var contextDescription = map[string]string{
	"":      "Unknown context, try updating your testkube cluster installation",
	"oss":   "Open Source Testkube",
	"cloud": "Testkube in Pro mode",
}
