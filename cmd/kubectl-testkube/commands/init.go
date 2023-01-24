package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInitCmd() *cobra.Command {
	var options HelmUpgradeOrInstalTestkubeOptions

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Install Helm chart registry in current kubectl context and update dependencies",
		Aliases: []string{"install"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("WELCOME TO")
			ui.Logo()

			err := HelmUpgradeOrInstalTestkube(options)
			ui.ExitOnError("Installing testkube", err)

			ui.Info(`To help improve the quality of Testkube, we collect anonymous basic telemetry data.  Head out to https://kubeshop.github.io/testkube/telemetry/ to read our policy or feel free to:`)

			ui.NL()
			ui.ShellCommand("disable telemetry by typing", "testkube disable telemetry")
			ui.Info(" Happy Testing! 🚀")
			ui.NL()

		},
	}

	PopulateUpgradeInstallFlags(cmd, &options)

	return cmd
}
