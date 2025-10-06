package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

func NewUpgradeCmd() *cobra.Command {
	var options common.HelmOptions
	var dockerContainerName string

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Helm chart, install dependencies and run migrations",
		Aliases: []string{"update"},
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)
			ui.NL()

			common.ProcessMasterFlags(cmd, &options, &cfg)

			// set to cloud context explicitly when user pass agent key and store the key later
			if options.Master.AgentToken != "" {
				cfg.CloudContext.AgentKey = options.Master.AgentToken
				cfg.ContextType = config.ContextTypeCloud
			}

			if cmd.Flags().Changed("docker-container") {
				cfg.CloudContext.DockerContainerName = dockerContainerName
			}

			if !options.NoConfirm {
				ui.Warn("This will upgrade Testkube to the latest version. This may take a few minutes.")
				if cfg.CloudContext.DockerContainerName != "" {
					ui.Warn("Please be sure you have Docker service running before continuing and can run containers in privileged mode!")

					dockerInfo, cliErr := common.RunDockerCommand([]string{"info"})
					if cliErr != nil {
						common.HandleCLIError(cliErr)
					}

					ui.Alert("Current docker info:", dockerInfo)
					ui.NL()
				} else {
					ui.Warn("Please be sure you're on valid kubectl context before continuing!")

					currentContext, cliErr := common.GetCurrentKubernetesContext()
					common.HandleCLIError(cliErr)

					ui.ExitOnError("getting current context", err)
					ui.Alert("Current kubectl context:", currentContext)
					ui.NL()
				}

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
					ui.Errf("Upgrade cancelled")
					return
				}
			}

			if cfg.ContextType == config.ContextTypeCloud {
				ui.Info("Testkube Pro agent upgrade started")
				if cfg.CloudContext.DockerContainerName != "" {
					latestVersion, errLatestVersion := common.GetLatestVersion()
					ui.ExitOnError("Getting latest version", errLatestVersion)
					err = common.DockerUpgradeTestkubeAgent(options, latestVersion, cfg)
				} else {
					err = common.HelmUpgradeOrInstallTestkubeAgent(options, cfg, false)
				}
				ui.ExitOnError("Upgrading Testkube Pro Agent", err)
				err = common.PopulateAgentDataToContext(options, cfg)
				ui.ExitOnError("Storing agent data in context", err)
			} else {
				ui.Info("Updating Testkube")

				if cliErr := common.HelmUpgradeOrInstallTestkube(options); cliErr != nil {
					cliErr.Print()
					os.Exit(1)
				}
			}

		},
	}

	common.PopulateHelmFlags(cmd, &options)
	common.PopulateMasterFlags(cmd, &options, false)

	cmd.Flags().StringVar(&dockerContainerName, "docker-container", "testkube-agent", "Docker container name for Testkube Docker Agent")

	return cmd
}
