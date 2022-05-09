package commands

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewInstallCmd() *cobra.Command {
	var options HelmUpgradeOrInstalTestkubeOptions

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Helm chart registry in current kubectl context and update dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			err := HelmUpgradeOrInstalTestkube(options)
			ui.ExitOnError("Installing testkube", err)
		},
	}

	PopulateUpgradeInstallFlags(cmd, &options)

	return cmd
}
