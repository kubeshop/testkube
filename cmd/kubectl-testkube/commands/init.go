package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils/text"
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

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)
			ui.NL()

			// set to cloud context explicitly when user pass agent key and store the key later
			if options.CloudAgentToken != "" {
				cfg.CloudContext.AgentKey = options.CloudAgentToken
				cfg.ContextType = config.ContextTypeCloud
				options.CloudUris = common.NewCloudUris(options.CloudRootDomain)
			}

			if !options.NoConfirm {
				ui.Warn("This will install Testkube to the latest version. This may take a few minutes.")
				ui.Warn("Please be sure you're on valid kubectl context before continuing!")
				ui.NL()

				currentContext, err := common.GetCurrentKubernetesContext()
				ui.ExitOnError("getting current context", err)
				ui.Alert("Current kubectl context:", currentContext)
				ui.NL()

				if ui.IsVerbose() && cfg.ContextType == config.ContextTypeCloud {
					ui.Info("Your Testkube is in 'cloud' mode with following context")
					ui.InfoGrid(map[string]string{
						"Agent Key": text.Obfuscate(cfg.CloudContext.AgentKey),
						"Agent URI": cfg.CloudContext.AgentUri,
					})
					ui.NL()
				}

				ok := ui.Confirm("Do you want to continue?")
				if !ok {
					ui.Errf("Testkube installation cancelled")
					return
				}
			}

			err = common.HelmUpgradeOrInstalTestkube(options)
			ui.ExitOnError("Installing testkube", err)

			if cfg.ContextType == config.ContextTypeCloud {
				err = common.PopulateAgentDataToContext(options, cfg)
				ui.ExitOnError("Storing agent data in context", err)
			} else {
				ui.Info(`To help improve the quality of Testkube, we collect anonymous basic telemetry data.  Head out to https://docs.testkube.io/articles/telemetry to read our policy or feel free to:`)

				ui.NL()
				ui.ShellCommand("disable telemetry by typing", "testkube disable telemetry")
				ui.NL()
			}

			ui.Info(" Happy Testing! 🚀")
			ui.NL()

		},
	}

	common.PopulateHelmFlags(cmd, &options)

	return cmd
}
