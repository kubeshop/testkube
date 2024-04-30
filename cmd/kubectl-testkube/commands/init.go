package commands

import (
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	standaloneAgentProfile = "standalone-agent"
	demoProfile            = "demo"
	agentProfile           = "agent"

	standaloneInstallationName = "Testkube Standalone(OSS)"
	demoInstallationName       = "Testkube Enterprise On-Premise"
	agentInstallationName      = "Testkube Agent"
)

func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init <profile>",
		Aliases: []string{"g"},
		Short:   "Init Testkube profiles(" + standaloneAgentProfile + "|" + demoProfile + "|" + agentProfile + ")",
		Long: "Init installs the Testkube in your cluster as follows:\n" +
			"\t" + standaloneAgentProfile + " -> " + standaloneInstallationName + "\n" +
			"\t" + demoProfile + " -> " + demoInstallationName + "\n" +
			"\t" + agentProfile + " -> " + agentInstallationName,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
	}

	cmd.AddCommand(NewInitCmdStandalone())
	cmd.AddCommand(NewInitCmdDemo())

	return cmd
}

func NewInitCmdStandalone() *cobra.Command {
	var options common.HelmOptions

	cmd := &cobra.Command{
		Use:     standaloneAgentProfile,
		Short:   "Install " + standaloneInstallationName + " Helm chart registry in current kubectl context and update dependencies",
		Aliases: []string{"install"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("WELCOME TO")
			ui.Logo()

			ui.NL()

			if !isContextApproved(options.NoConfirm, standaloneInstallationName) {
				return
			}

			common.ProcessMasterFlags(cmd, &options, nil)

			err := common.HelmUpgradeOrInstalTestkube(options)
			ui.ExitOnError("Installing "+standaloneInstallationName, err)

			ui.Info(`To help improve the quality of Testkube, we collect anonymous basic telemetry data. Head out to https://docs.testkube.io/articles/telemetry to read our policy or feel free to:`)

			ui.NL()
			ui.ShellCommand("disable telemetry by typing", "testkube disable telemetry")
			ui.NL()

			ui.Info(" Happy Testing! 🚀")
			ui.NL()

		},
	}

	common.PopulateHelmFlags(cmd, &options)
	common.PopulateMasterFlags(cmd, &options)

	return cmd
}

func NewInitCmdDemo() *cobra.Command {
	var noConfirm, dryRun bool
	var license, namespace string

	cmd := &cobra.Command{
		Use:     demoProfile,
		Short:   "Install " + demoInstallationName + " Helm chart registry in current kubectl context and update dependencies",
		Aliases: []string{"on-premise", "on-prem", "enterprise"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("WELCOME TO")
			ui.Logo()

			ui.NL()
			if license == "" {
				ui.Warn("License key is required to install " + demoInstallationName)
				return
			}

			if !isContextApproved(noConfirm, demoInstallationName) {
				return
			}

			err := helmInstallDemo(license, namespace, dryRun)
			ui.ExitOnError("Installing "+demoInstallationName, err)

			ui.Info(`To help improve the quality of Testkube, we collect anonymous basic telemetry data. Head out to https://docs.testkube.io/articles/telemetry to read our policy or feel free to:`)

			ui.NL()
			ui.ShellCommand("disable telemetry by typing", "testkube disable telemetry")
			ui.NL()

			ui.Info(" Happy Testing! 🚀")
			ui.NL()

		},
	}

	cmd.Flags().BoolVarP(&noConfirm, "no-confirm", "y", false, "Skip confirmation")
	cmd.Flags().StringVarP(&license, "license", "l", "", "License key")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "Dry run")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "testkube-enterprise", "Namespace to install "+demoInstallationName)

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
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return err
	}
	ui.Info("Helm installing " + demoInstallationName)
	args := []string{"repo", "add", "testkubeenterprise", "https://kubeshop.github.io/testkube-cloud-charts"}
	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: dryRun})
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (kubeshop) already exists, please specify a different name") {
		ui.WarnOnError("adding testkube repo", err)
	}

	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: []string{"repo", "update"}, DryRun: dryRun})
	ui.ExitOnError("updating helm repositories", err)

	//helm upgrade --install --create-namespace --namespace=testkube testkube -f=profiles/values.demo.yaml . --set global.enterpriseLicense=$LICENSE
	args = []string{"upgrade", "--install",
		"--create-namespace", "--namespace", namespace,
		"--set", "global.enterpriseLicense=" + license,
		"--values", "https://raw.githubusercontent.com/kubeshop/testkube-cloud-charts/main/charts/testkube-enterprise/profiles/values.demo.yaml",
		"testkube-enterprise", "testkubeenterprise/testkube-enterprise"}
	out, err := process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: dryRun})
	ui.ExitOnError("installing "+demoInstallationName, err)

	cfg, err := config.Load()
	if err == nil {
		cfg.EnterpriseNamespace = namespace
		err = config.Save(cfg)
		ui.ExitOnError("saving config file", err)
	}

	ui.Debug("Helm run command: ")
	ui.Debug(helmPath, args...)

	ui.Debug("Helm command output: ", string(out))
	return nil

}
