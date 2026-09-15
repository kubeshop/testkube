package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	licensevalidator "github.com/kubeshop/testkube/pkg/diagnostics/validators/license"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	defaultNamespace       = "testkube"
	standaloneAgentProfile = "standalone-agent"
	demoProfile            = "demo"
	demoValuesUrl          = "https://raw.githubusercontent.com/kubeshop/testkube-cloud-charts/main/charts/testkube-enterprise/profiles/values.demo.v2.yaml"
	agentProfile           = "agent"

	standaloneInstallationName = "Testkube OSS"
	demoInstallationName       = "Testkube On-Prem demo"
	agentInstallationName      = "Testkube Agent"
)

func NewInitCmd() *cobra.Command {
	var export bool
	standaloneCmd := NewInitCmdStandalone()

	cmd := &cobra.Command{
		Use:     "init <profile>",
		Aliases: []string{"g"},
		Short:   "Init Testkube profiles(" + standaloneAgentProfile + "|" + demoProfile + "|" + agentProfile + ")",
		Long: "Init installs the Testkube in your cluster as follows:\n" +
			"\t" + standaloneAgentProfile + " -> " + standaloneInstallationName + "\n" +
			"\t" + demoProfile + " -> " + demoInstallationName + "\n" +
			"\t" + agentProfile + " -> " + agentInstallationName,
		Run: func(cmd *cobra.Command, args []string) {
			if export {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrInvalidRuntimeParameter,
					"Export is unavailable for this profile",
					"Drop the '--export' flag, it is only supported by the "+demoProfile+" profile",
					errors.New("export is unavailable for this profile"),
				))
			}
			standaloneCmd.Run(cmd, args)
		},
	}

	cmd.AddCommand(standaloneCmd)
	cmd.AddCommand(NewInitCmdDemo())
	cmd.AddCommand(pro.NewInitCmd())
	cmd.Flags().BoolVarP(&export, "export", "", false, "Export the values.yaml")

	return cmd
}

func NewInitCmdStandalone() *cobra.Command {
	var export bool
	var options common.HelmOptions
	var setOptions, argOptions map[string]string

	cmd := &cobra.Command{
		Use:     standaloneAgentProfile,
		Short:   "Install " + standaloneInstallationName + " in your current context",
		Aliases: []string{"oss", "standalone"},
		Run: func(cmd *cobra.Command, args []string) {
			if export {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrInvalidRuntimeParameter,
					"Export is unavailable for this profile",
					"Drop the '--export' flag, it is only supported by the "+demoProfile+" profile",
					errors.New("export is unavailable for this profile"),
				))
			}

			ui.Logo()
			ui.Info("Welcome to the installer for " + standaloneInstallationName + ".")
			ui.NL()
			common.ShowOperatorDeprecationWarning("Testkube API Server", options.NoCRDs)

			if !isContextApproved(options.NoConfirm, standaloneInstallationName) {
				return
			}

			common.ProcessMasterFlags(cmd, &options, nil)
			options.SetOptions = setOptions
			options.ArgOptions = argOptions
			ui.NL()
			ui.H2("Running Helm command...")
			ui.NL()
			common.HandleCLIError(common.HelmUpgradeOrInstallTestkube(options))

			ui.Info(`To help improve the quality of Testkube, we collect anonymous basic telemetry data. Head out to https://docs.testkube.io/articles/telemetry to read our policy or feel free to:`)

			ui.NL()
			ui.ShellCommand("disable telemetry by typing", "testkube disable telemetry")
			ui.NL()

			ui.Info(" Happy Testing! 🚀")
			ui.NL()

		},
	}

	cmd.Flags().BoolVarP(&export, "export", "", false, "Export the values.yaml")
	cmd.Flags().StringToStringVarP(&setOptions, "helm-set", "", nil, "helm set option in form of key=value")
	cmd.Flags().StringToStringVarP(&argOptions, "helm-arg", "", nil, "helm arg option in form of key=value")
	common.PopulateHelmFlags(cmd, &options)
	common.PopulateMasterFlags(cmd, &options, false)

	return cmd
}

func NewInitCmdDemo() *cobra.Command {
	var noConfirm, dryRun, export bool
	var license, namespace string
	var setOptions, argOptions map[string]string

	cmd := &cobra.Command{
		Use:     demoProfile,
		Short:   "Install " + demoInstallationName + " in your current context",
		Aliases: []string{"on-premise-demo", "on-prem-demo", "enterprise-demo"},
		Run: func(cmd *cobra.Command, args []string) {
			if export {
				exitOnExportError := func(err error) {
					if err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrValuesExportFailed,
							"Error exporting the installation values",
							"Check your internet connection and access to "+demoValuesUrl,
							err,
						))
					}
				}

				req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, demoValuesUrl, nil)
				exitOnExportError(err)
				valuesResp, err := http.DefaultClient.Do(req)
				exitOnExportError(err)
				defer valuesResp.Body.Close()
				valuesBytes, err := io.ReadAll(valuesResp.Body)
				exitOnExportError(err)
				_, err = fmt.Println(string(valuesBytes))
				exitOnExportError(err)
				return
			}

			ui.Logo()
			ui.Info("Welcome to the installer for " + demoInstallationName + ".")
			ui.NL()

			cfg, err := config.Load()
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					common.ConfigFileHint,
					err,
				))
			}

			sendTelemetry(cmd, cfg, license, "installation launched")

			ui.NL()
			ui.H2("Running Kubectl command...")
			ui.NL()
			kubecontext, cliErr := common.GetCurrentKubernetesContext()
			exitOnInstallError(cmd, cfg, "kubeconfig not found", "kubeconfig_not_found", license, cliErr)
			sendTelemetry(cmd, cfg, license, "kubeconfig found")

			if namespace == "" {
				if noConfirm {
					namespace = defaultNamespace
				} else {
					response, err := pterm.DefaultInteractiveTextInput.WithDefaultValue("testkube").Show("Enter namespace for this installation")
					if err != nil {
						exitOnInstallError(cmd, cfg, "input namespace", "reading_namespace_failed", license,
							common.NewCLIError(common.TKErrConfigInitFailed, "Error reading namespace from console",
								"Check does the current system have a console, or pass the namespace with the '--namespace' flag", err))
					}
					namespace = response
				}
			}
			sendTelemetry(cmd, cfg, license, "namespace found")
			if license == "" {
				response, err := pterm.DefaultInteractiveTextInput.Show("Enter license key")
				if err != nil {
					exitOnInstallError(cmd, cfg, "input license", "reading_license_failed", license,
						common.NewCLIError(common.TKErrConfigInitFailed, "Error reading license key from console",
							"Check does the current system have a console, or pass the key with the '--license' flag", err))
				}
				license = strings.TrimSpace(response)
			}
			sendTelemetry(cmd, cfg, license, "license found")

			licenseName := ""
			resp, validateErr := licensevalidator.NewClient().ValidateLicense(licensevalidator.LicenseRequest{License: license})
			switch {
			case validateErr != nil:
				ui.Debug("license validation request failed, continuing", validateErr.Error())
			case resp != nil && !resp.Valid:
				exitOnInstallError(cmd, cfg, "license validation", "install_license_invalid", license, common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Invalid license key",
					"Check that your license key is correct and active, or request a trial license at https://testkube.io/download",
					fmt.Errorf("license validation failed: code=%q %s", resp.Code, resp.Message),
				))
			case resp != nil:
				licenseName = resp.License.Name
			}
			sendTelemetry(cmd, cfg, license, "license validated", licenseName)

			ui.NL()
			ui.Warn("Installation is about to start and may take a several minutes:")
			ui.NL()
			ui.Warn("- This install typically works best in a local cluster.")
			ui.Warn("- Testkube will be installed in the " + kubecontext + " context.")
			ui.Warn("- Testkube services will be applied to the " + namespace + " namespace.")
			ui.Warn("- Testkube CRDs and cluster roles will be applied to your cluster.")
			ui.NL()

			if !noConfirm {
				if ok := ui.Confirm("Do you want to continue"); !ok {
					sendErrTelemetry(cmd, cfg, "install_cancelled", license, "user install confirmation",
						errors.New("user cancelled installation"))
					return
				}
			}

			spinner := ui.NewSpinner("Running Kubectl command...")
			sendTelemetry(cmd, cfg, license, "installing started", licenseName)
			options := common.HelmOptions{
				Namespace:     namespace,
				LicenseKey:    license,
				DemoValuesURL: demoValuesUrl,
				DryRun:        dryRun,
			}

			cliErr = common.CleanExistingCompletedMigrationJobs(options.Namespace)
			if cliErr != nil {
				spinner.Fail("Failed to install Testkube On-Prem Demo")
				exitOnInstallError(cmd, cfg, "installing", "install_failed", license, cliErr)
			}

			spinner.Success()
			spinner = ui.NewSpinner("Running Helm command...")
			options.SetOptions = setOptions
			options.ArgOptions = argOptions

			runnerSecretKey, cliErr := common.ResolveDemoAgentSecretKey(options.Namespace, options.DryRun)
			if cliErr != nil {
				spinner.Fail("Failed to install Testkube On-Prem Demo")
				exitOnInstallError(cmd, cfg, "resolving agent key", "install_failed", license, cliErr)
			}

			cliErr = common.HelmUpgradeOrInstallTestkubeOnPremDemo(options, runnerSecretKey)
			if cliErr != nil {
				spinner.Fail("Failed to install Testkube On-Prem Demo")
				exitOnInstallError(cmd, cfg, "installing", "install_failed", license, cliErr)
			}
			spinner.Success()

			spinner = ui.NewSpinner("Installing Testkube Runner...")
			cliErr = common.HelmUpgradeOrInstallTestkubeOnPremDemoRunner(options, runnerSecretKey)
			if cliErr != nil {
				spinner.Fail("Failed to install Testkube Runner")
				exitOnInstallError(cmd, cfg, "installing runner", "install_runner_failed", license, cliErr)
			}
			spinner.Success()

			sendTelemetry(cmd, cfg, license, "installing finished", licenseName)

			cfg.Namespace = namespace
			err = config.Save(cfg)
			if err != nil {
				ui.Debug("Cannot save config")
			}

			ui.Info("The default admin credentials are: admin@example.com / password")
			ui.Info("Make sure to copy these credentials now as you will not be able to see this again.")
			ui.NL()
			ok := ui.Confirm("Do you want to continue?")
			if !ok {
				return
			}

			sendTelemetry(cmd, cfg, license, "user confirmed proceeding", licenseName)

			ui.Info("You can use `testkube dashboard` to access Testkube without exposing services.")
			ui.NL()

			if ok := ui.Confirm("Do you want to open the dashboard?"); !ok {
				sendTelemetry(cmd, cfg, license, "skipping dashboard", licenseName)
				return
			}
			sendTelemetry(cmd, cfg, license, "opening dashboard", licenseName)
			cfg, err = config.Load()
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					common.ConfigFileHint,
					err,
				))
			}

			ui.NL()
			ui.H2("Launching web browser...")
			ui.NL()
			openOnPremDashboard(cmd, cfg, false, false, license)
		},
	}

	cmd.Flags().BoolVarP(&export, "export", "", false, "Export the values.yaml")
	cmd.Flags().BoolVarP(&noConfirm, "no-confirm", "y", false, "Skip confirmation")
	cmd.Flags().StringVarP(&license, "license", "l", "", "License key")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "Dry run")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to install "+demoInstallationName)
	cmd.Flags().StringToStringVarP(&setOptions, "helm-set", "", nil, "helm set option in form of key=value")
	cmd.Flags().StringToStringVarP(&argOptions, "helm-arg", "", nil, "helm arg option in form of key=value")

	return cmd
}

func isContextApproved(isNoConfirm bool, installedComponent string) bool {
	if !isNoConfirm {
		ui.Warn("This will install " + installedComponent + " to the latest version. This may take a few minutes.")
		ui.Warn("Please be sure you're on valid kubectl context before continuing!")
		ui.NL()

		currentContext, err := common.GetCurrentKubernetesContext()
		common.HandleCLIError(err)

		ui.Alert("Current kubectl context:", currentContext)
		ui.NL()

		ok := ui.Confirm("Do you want to continue?")
		if !ok {
			ui.Errf("Installation cancelled")
			return false
		}
	}
	return true
}

// exitOnInstallError reports the failure with the license aware telemetry this
// installer collects, then prints the error details and terminates the command.
// It is a no-op for a nil error, so it can be called directly on a helper's
// returned *CLIError.
func exitOnInstallError(cmd *cobra.Command, clientCfg config.Data, step, errType, license string, cliErr *common.CLIError) {
	if cliErr == nil {
		return
	}

	if clientCfg.TelemetryEnabled {
		cliErr.AddTelemetry(cmd, step, errType, license)
		_, _ = handleCLIErrorTelemetry(common.Version, cliErr)
	}

	common.HandleCLIError(cliErr)
}

func sendErrTelemetry(cmd *cobra.Command, clientCfg config.Data, errType, license, step string, errorLogs error) {
	errorStackTrace := fmt.Sprintf("%+v", errorLogs)
	if clientCfg.TelemetryEnabled {
		out, err := telemetry.SendCmdErrorEventWithLicense(cmd, common.Version, errType, errorStackTrace, license, step, "")
		if ui.Verbose && err != nil {
			ui.Err(err)
		}

		ui.Debug("telemetry send event response", out)
	}
}

func sendTelemetry(cmd *cobra.Command, clientCfg config.Data, license, step string, userIDOverride ...string) {
	if clientCfg.TelemetryEnabled {
		userID := common.TelemetryUserID(cmd, &clientCfg)
		if len(userIDOverride) > 0 && userIDOverride[0] != "" {
			userID = userIDOverride[0]
		}
		out, err := telemetry.SendCmdWithLicenseEvent(cmd, common.Version, userID, license, step)
		if ui.Verbose && err != nil {
			ui.Err(err)
		}
		ui.Debug("telemetry send event response", out)
	}
}
