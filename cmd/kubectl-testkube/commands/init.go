package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInitCmd() *cobra.Command {
	var options common.HelmOptions

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Install Helm chart registry in current kubectl context and update dependencies",
		Aliases: []string{"install"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("WELCOME TO")
			ui.Logo()

			ui.NL()

			if !options.NoConfirm {
				ui.Warn("This will install Testkube to the latest version. This may take a few minutes.")
				ui.Warn("Please be sure you're on valid kubectl context before continuing!")
				ui.NL()

				currentContext, err := common.GetCurrentKubernetesContext()
				ui.ExitOnError("getting current context", err)
				ui.Alert("Current kubectl context:", currentContext)
				ui.NL()

				ok := ui.Confirm("Do you want to continue?")
				if !ok {
					ui.Errf("Testkube installation cancelled")
					return
				}
			}

			common.ProcessMasterFlags(cmd, &options, nil)

			err := common.HelmUpgradeOrInstalTestkube(options)
			ui.ExitOnError("Installing testkube", err)

			ui.Info(`To help improve the quality of Testkube, we collect anonymous basic telemetry data.  Head out to https://docs.testkube.io/articles/telemetry to read our policy or feel free to:`)

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
