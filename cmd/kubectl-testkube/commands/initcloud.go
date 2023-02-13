package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInitCloudCmd() *cobra.Command {
	var options HelmUpgradeOrInstalTestkubeOptions

	cmd := &cobra.Command{
		Use:     "init-cloud",
		Short:   "Installs Testkube as an Agent",
		Aliases: []string{"init-agent", "install-agent"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("WELCOME TO")
			ui.Logo()

			if options.AgentUri == "" || options.AgentKey == "" {
				ui.Failf("Both 'agent-uri' and 'agent-key' must be provided")
			}

			cfg, err := config.Load()
			ui.ExitOnError("Loading config", err)

			err = HelmUpgradeOrInstalTestkubeCloud(options, cfg)
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
