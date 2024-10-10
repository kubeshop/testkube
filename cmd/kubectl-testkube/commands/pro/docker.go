package pro

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDockerCmd() *cobra.Command {
	var noLogin bool // ignore ask for login
	var containerName, dockerImage string
	var options common.HelmOptions

	cmd := &cobra.Command{
		Use:     "docker",
		Short:   "Run Testkube Docker Agent and connect to Testkube Pro environment",
		Aliases: []string{"da", "docker-agent"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("WELCOME TO")
			ui.Logo()

			cfg, err := config.Load()
			if err != nil {
				cliErr := common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					"Check is the Testkube config file (~/.testkube/config.json) accessible and has right permissions",
					err,
				)
				cliErr.Print()
				os.Exit(1)
			}
			ui.NL()

			common.ProcessMasterFlags(cmd, &options, &cfg)

			sendAttemptTelemetry(cmd, cfg)

			if !options.NoConfirm {
				ui.Warn("This will run Testkube Docker Agent latest version. This may take a few minutes.")
				ui.Warn("Please be sure you have Docker service running before continuing!")
				ui.NL()

				dockerInfo, cliErr := common.RunDockerCommand([]string{"info"})
				if cliErr != nil {
					sendErrTelemetry(cmd, cfg, "docker_info", err)
					common.HandleCLIError(cliErr)
				}
				ui.Alert("Current docker info:", dockerInfo)
				ui.NL()

				ok := ui.Confirm("Do you want to continue?")
				if !ok {
					ui.Errf("Testkube Docker Agent running cancelled")
					sendErrTelemetry(cmd, cfg, "user_cancel", err)
					return
				}
			}

			spinner := ui.NewSpinner("Running Testkube Docker Agent")
			if cliErr := common.DockerRunTestkubeAgent(options, cfg, containerName, dockerImage); cliErr != nil {
				spinner.Fail()
				sendErrTelemetry(cmd, cfg, "docker_run", cliErr)
				common.HandleCLIError(cliErr)
			}

			spinner.Success()

			ui.NL()

			if noLogin {
				ui.Alert("Saving Testkube CLI Pro context, you need to authorize CLI through `testkube set context` later")
				common.PopulateCloudConfig(cfg, "", &options)
				ui.Info(" Happy Testing! 🚀")
				ui.NL()
				return
			}

			ui.H2("Saving Testkube CLI Pro context")
			var token, refreshToken string
			if !common.IsUserLoggedIn(cfg, options) {
				token, refreshToken, err = common.LoginUser(options.Master.URIs.Auth)
				sendErrTelemetry(cmd, cfg, "login", err)
				ui.ExitOnError("user login", err)
			}
			err = common.PopulateLoginDataToContext(options.Master.OrgId, options.Master.EnvId, token, refreshToken, options, cfg)
			if err != nil {
				sendErrTelemetry(cmd, cfg, "setting_context", err)
				ui.ExitOnError("Setting Pro environment context", err)
			}
			ui.Info(" Happy Testing! 🚀")
			ui.NL()
		},
	}

	common.PopulateMasterFlags(cmd, &options)

	cmd.Flags().BoolVarP(&noLogin, "no-login", "", false, "Ignore login prompt, set existing token later by `testkube set context`")
	cmd.Flags().StringVar(&containerName, "container-name", "testkube-agent", "container name for Testkube Docker Agent")
	cmd.Flags().StringVar(&dockerImage, "docker-image", "kubeshop/testkube-agent", "docker image for Testkube Docker Agent")

	return cmd
}
