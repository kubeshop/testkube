package commands

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	defaultNamespace       = "testkube"
	standaloneAgentProfile = "standalone-agent"
	demoProfile            = "demo"
	demoValuesUrl          = "https://raw.githubusercontent.com/kubeshop/testkube-cloud-charts/main/charts/testkube-enterprise/profiles/values.demo.yaml"
	agentProfile           = "agent"

	standaloneInstallationName = "Testkube OSS"
	demoInstallationName       = "Testkube On-Prem demo"
	agentInstallationName      = "Testkube Agent"
	licenseFormat              = "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX-V3"
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
				ui.Failf("export is unavailable for this profile")
				return
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
				ui.Failf("export is unavailable for this profile")
				return
			}

			ui.Logo()
			ui.Info("Welcome to the installer for " + standaloneInstallationName + ".")
			ui.NL()

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
				valuesResp, err := http.Get(demoValuesUrl)
				ui.ExitOnError("cannot fetch values", err)
				defer valuesResp.Body.Close()
				valuesBytes, err := io.ReadAll(valuesResp.Body)
				ui.ExitOnError("cannot fetch values", err)
				values := string(valuesBytes)
				_, err = fmt.Println(values)
				ui.ExitOnError("cannot print values", err)
				return
			}

			ui.Logo()
			ui.Info("Welcome to the installer for " + demoInstallationName + ".")
			ui.NL()

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			sendTelemetry(cmd, cfg, license, "installation launched")

			ui.NL()
			ui.H2("Running Kubectl command...")
			ui.NL()
			kubecontext, cliErr := common.GetCurrentKubernetesContext()
			if cliErr != nil {
				if cfg.TelemetryEnabled {
					cliErr.AddTelemetry(cmd, "kubeconfig not found", "kubeconfig_not_found", license)
					_, _ = telemetry.HandleCLIErrorTelemetry(common.Version, cliErr)
				}
				common.HandleCLIError(cliErr)
			}
			sendTelemetry(cmd, cfg, license, "kubeconfig found")

			if namespace == "" {
				if noConfirm {
					namespace = defaultNamespace
				} else {
					response, err := pterm.DefaultInteractiveTextInput.WithDefaultValue("testkube").Show("Enter namespace for this installation")
					if err != nil {
						cliErr = common.NewCLIError(common.TKErrConfigInitFailed, "Error reading namespace from console", "Check does the current system have a console.", err)
						if cfg.TelemetryEnabled {
							cliErr.AddTelemetry(cmd, "input namespace", "reading_namespace_failed", license)
							_, _ = telemetry.HandleCLIErrorTelemetry(common.Version, cliErr)
						}
						common.HandleCLIError(cliErr)
					}
					namespace = response
				}
			}
			sendTelemetry(cmd, cfg, license, "namespace found")
			if license == "" {
				response, err := pterm.DefaultInteractiveTextInput.Show("Enter license key")
				if err != nil {
					cliErr = common.NewCLIError(common.TKErrConfigInitFailed, "Error reading namespace from console", "Check does the current system have a console.", err)
					if cfg.TelemetryEnabled {
						cliErr.AddTelemetry(cmd, "input license", "reading_license_failed", license)
						_, _ = telemetry.HandleCLIErrorTelemetry(common.Version, cliErr)
					}
					common.HandleCLIError(cliErr)
				}
				license = strings.TrimSpace(response)
			}
			sendTelemetry(cmd, cfg, license, "license found")

			hasExpectedLength := len(license) == len(licenseFormat)
			hasExpectedPrefix := strings.HasPrefix(license, "key/")
			valid := hasExpectedLength || hasExpectedPrefix
			if !valid {
				cliErr = common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Invalid license key",
					"Check have you correctly inputted your license key or request a trial license at https://testkube.io/download",
					fmt.Errorf("license key is missing or has invalid format"),
				)
				if cfg.TelemetryEnabled {
					cliErr.AddTelemetry(cmd, "license validation", "install_license_invalid", license)
					_, _ = telemetry.HandleCLIErrorTelemetry(common.Version, cliErr)
				}
				common.HandleCLIError(cliErr)
			}
			sendTelemetry(cmd, cfg, license, "license validated")

			ui.NL()
			ui.Warn("Installation is about to start and may take a several minutes:")
			ui.NL()
			ui.Warn("- Testkube will be installed in the " + kubecontext + " context.")
			ui.Warn("- Testkube services will be applied to the " + namespace + " namespace.")
			ui.Warn("- Testkube CRDs and cluster roles will be applied to your cluster.")
			ui.NL()

			if !noConfirm {
				if ok := ui.Confirm("Do you want to continue"); !ok {
					sendErrTelemetry(cmd, cfg, "install_cancelled", license, "user install confirmation", err)
					return
				}
			}

			spinner := ui.NewSpinner("Running Kubectl command...")
			sendTelemetry(cmd, cfg, license, "installing started")
			options := common.HelmOptions{
				Namespace:     namespace,
				LicenseKey:    license,
				DemoValuesURL: demoValuesUrl,
				DryRun:        dryRun,
			}

			cliErr = common.CleanExistingCompletedMigrationJobs(options.Namespace)
			if cliErr != nil {
				spinner.Fail("Failed to install Testkube On-Prem Demo")
				if cfg.TelemetryEnabled {
					cliErr.AddTelemetry(cmd, "installing", "install_failed", license)
					_, _ = telemetry.HandleCLIErrorTelemetry(common.Version, cliErr)
				}
				common.HandleCLIError(cliErr)
			}

			spinner.Success()
			spinner = ui.NewSpinner("Running Helm command...")
			options.SetOptions = setOptions
			options.ArgOptions = argOptions
			cliErr = common.HelmUpgradeOrInstallTestkubeOnPremDemo(options)
			if cliErr != nil {
				spinner.Fail("Failed to install Testkube On-Prem Demo")
				if cfg.TelemetryEnabled {
					cliErr.AddTelemetry(cmd, "installing", "install_failed", license)
					_, _ = telemetry.HandleCLIErrorTelemetry(common.Version, cliErr)
				}
				common.HandleCLIError(cliErr)
			}
			spinner.Success()

			sendTelemetry(cmd, cfg, license, "installing finished")

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

			sendTelemetry(cmd, cfg, license, "user confirmed proceeding")

			ui.Info("You can use `testkube dashboard` to access Testkube without exposing services.")
			ui.NL()

			if ok := ui.Confirm("Do you want to open the dashboard?"); !ok {
				sendTelemetry(cmd, cfg, license, "skipping dashboard")
				return
			}
			sendTelemetry(cmd, cfg, license, "opening dashboard")
			cfg, err = config.Load()
			ui.ExitOnError("Cannot open dashboard", err)

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

func sendTelemetry(cmd *cobra.Command, clientCfg config.Data, license, step string) {
	if clientCfg.TelemetryEnabled {
		out, err := telemetry.SendCmdWithLicenseEvent(cmd, common.Version, license, step)
		if ui.Verbose && err != nil {
			ui.Err(err)
		}
		ui.Debug("telemetry send event response", out)
	}
}
