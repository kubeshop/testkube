package commands

import (
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
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					common.ConfigFileHint,
					err,
				))
			}
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
				// Both upgrade helpers return *CLIError, so the result has to stay
				// typed: assigning a nil *CLIError to an error turns it into a
				// non-nil interface and reports a successful upgrade as a failure.
				var upgradeErr *common.CLIError
				if cfg.CloudContext.DockerContainerName != "" {
					latestVersion, errLatestVersion := common.GetLatestVersion()
					if errLatestVersion != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrLatestVersionFetchFailed,
							"Error getting the latest Testkube version",
							"Check your internet connection and access to github.com",
							errLatestVersion,
						))
					}
					upgradeErr = common.DockerUpgradeTestkubeAgent(options, latestVersion, cfg)
				} else {
					upgradeErr = common.HelmUpgradeOrInstallTestkubeAgent(options, cfg, false)
				}
				common.HandleCLIError(upgradeErr)

				if err = common.PopulateAgentDataToContext(options, cfg); err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrContextSaveFailed,
						"Error storing agent data in context",
						common.ConfigFileHint,
						err,
					))
				}
			} else {
				ui.Info("Updating Testkube")

				common.HandleCLIError(common.HelmUpgradeOrInstallTestkube(options))
			}

		},
	}

	common.PopulateHelmFlags(cmd, &options)
	common.PopulateMasterFlags(cmd, &options, false)

	cmd.Flags().StringVar(&dockerContainerName, "docker-container", "testkube-agent", "Docker container name for Testkube Docker Agent")

	return cmd
}
