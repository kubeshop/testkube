package commands

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/process"
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
	helpUrl                    = "https://testkubeworkspace.slack.com"
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

			err := common.HelmUpgradeOrInstalTestkube(options)
			ui.ExitOnError("Cannot install Testkube", err)

			ui.Info(`To help improve the quality of Testkube, we collect anonymous basic telemetry data. Head out to https://docs.testkube.io/articles/telemetry to read our policy or feel free to:`)

			ui.NL()
			ui.ShellCommand("disable telemetry by typing", "testkube disable telemetry")
			ui.NL()

			ui.Info(" Happy Testing! 🚀")
			ui.NL()

		},
	}

	cmd.Flags().BoolVarP(&export, "export", "", false, "Export the values.yaml")
	common.PopulateHelmFlags(cmd, &options)
	common.PopulateMasterFlags(cmd, &options)

	return cmd
}

func NewInitCmdDemo() *cobra.Command {
	var noConfirm, dryRun, export bool
	var license, namespace string

	cmd := &cobra.Command{
		Use:     demoProfile,
		Short:   "Install " + demoInstallationName + " in your current context",
		Aliases: []string{"on-premise-demo", "on-prem-demo", "enterprise-demo"},
		Run: func(cmd *cobra.Command, args []string) {
			if export {
				valuesResp, err := http.Get(demoValuesUrl)
				ui.ExitOnError("cannot fetch values", err)
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

			kubecontext, err := common.GetCurrentKubernetesContext()
			if err != nil {
				ui.Failf("kubeconfig not found")
			}

			if namespace == "" {
				if noConfirm {
					namespace = defaultNamespace
				} else {
					response, err := pterm.DefaultInteractiveTextInput.WithDefaultValue("testkube").Show("Enter namespace for this installation")
					namespace = response
					ui.ExitOnError("cannot read namespace", err)
				}
			}

			if license == "" {
				response, err := pterm.DefaultInteractiveTextInput.Show("Enter license key")
				license = strings.TrimSpace(response)
				ui.ExitOnError("cannot read license", err)
			}

			if len(license) != len(licenseFormat) {
				sendErrTelemetry(cmd, cfg, "install_license_malformed", license, err)
				ui.Failf("license malformed, expected license of format: " + licenseFormat)
			}

			ui.NL()
			ui.Warn("Installation is about to start and may take a several minutes:")
			ui.NL()
			ui.Warn("- Testkube will be installed in the " + kubecontext + " context.")
			ui.Warn("- Testkube services will be applied to the " + namespace + " namespace.")
			ui.Warn("- Testkube CRDs and cluster roles will be applied to your cluster.")
			ui.NL()

			if !noConfirm {
				if ok := ui.Confirm("Do you want to continue"); !ok {
					sendErrTelemetry(cmd, cfg, "install_cancelled", license, err)
					return
				}
			}

			sendAttemptTelemetry(cmd, cfg, license)
			err = helmInstallDemo(license, namespace, dryRun)
			if err != nil {
				sendErrTelemetry(cmd, cfg, "install_failed", license, err)
				ui.NL()
				ui.Info(fmt.Sprint(err))
				ui.NL()
				ui.Info("Let us help you!")
				ui.Info("Come say hi on Slack:", helpUrl)
				return
			}

			if err == nil {
				cfg.Namespace = namespace
				err = config.Save(cfg)
				if err != nil {
					ui.Debug("Cannot save config")
				}
			}

			ui.Info("Your initial admin credentials are: admin@example.com / password")
			ui.Info("Make sure to copy these credentials now as you will not be able to see this again.")
			ui.NL()
			ok := ui.Confirm("Do you want to continue?")

			ui.Info("You can use `testkube dashboard` to access Testkube without exposing services.")
			ui.NL()

			if !ok {
				return
			}

			if ok := ui.Confirm("Do you want to open the dashboard?"); ok {
				cfg, err := config.Load()
				ui.ExitOnError("Cannot open dashboard", err)
				openOnPremDashboard(nil, cfg, false)
			}
		},
	}

	cmd.Flags().BoolVarP(&export, "export", "", false, "Export the values.yaml")
	cmd.Flags().BoolVarP(&noConfirm, "no-confirm", "y", false, "Skip confirmation")
	cmd.Flags().StringVarP(&license, "license", "l", "", "License key")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "Dry run")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to install "+demoInstallationName)

	return cmd
}

func isContextApproved(isNoConfirm bool, installedComponent string) bool {
	if !isNoConfirm {
		ui.Warn("This will install " + installedComponent + " to the latest version. This may take a few minutes.")
		ui.Warn("Please be sure you're on valid kubectl context before continuing!")
		ui.NL()

		currentContext, err := common.GetCurrentKubernetesContext()
		ui.ExitOnError("getting current context", err)
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

func helmInstallDemo(license, namespace string, dryRun bool) error {
	spinner := ui.NewSpinner("Installing Testkube… ")

	helmPath, err := exec.LookPath("helm")
	if err != nil {
		spinner.Fail("Failed to install Testkube")
		return err
	}

	args := []string{"repo", "add", "testkubeenterprise", "https://kubeshop.github.io/testkube-cloud-charts"}
	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: dryRun})
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (kubeshop) already exists, please specify a different name") {
		spinner.Fail("Failed to install Testkube")
		return err
	}

	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: []string{"repo", "update"}, DryRun: dryRun})

	if err != nil {
		spinner.Fail("Failed to install Testkube")
		return err
	}

	args = []string{"upgrade", "--install",
		"--create-namespace",
		"--namespace", namespace,
		"--set", "global.enterpriseLicenseKey=" + license,
		"--values", demoValuesUrl,
		"--wait",
		"testkube", "testkubeenterprise/testkube-enterprise"}

	ui.Debug("Helm command: ", helmPath+" "+strings.Join(args, " "))

	out, err := process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: dryRun})
	if err != nil {
		spinner.Fail("Failed to install Testkube")
		return err
	}
	spinner.Success()

	ui.Debug("Helm command output: ", string(out))
	ui.NL()
	return nil
}

func sendErrTelemetry(cmd *cobra.Command, clientCfg config.Data, errType, license string, errorLogs error) {
	errorStackTrace := fmt.Sprintf("%+v", errorLogs)
	if clientCfg.TelemetryEnabled {
		out, err := telemetry.SendCmdErrorEventWithLicense(cmd, common.Version, errType, license, errorStackTrace)
		if ui.Verbose && err != nil {
			ui.Err(err)
		}

		ui.Debug("telemetry send event response", out)
	}
}

func sendAttemptTelemetry(cmd *cobra.Command, clientCfg config.Data, license string) {
	if clientCfg.TelemetryEnabled {
		out, err := telemetry.SendCmdAttempWithLicenseEvent(cmd, common.Version, license)
		if ui.Verbose && err != nil {
			ui.Err(err)
		}
		ui.Debug("telemetry send event response", out)
	}
}
