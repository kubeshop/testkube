package docker

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	StableReleasePlaceholder = "<latest-stable-release>"
)

func NewInitCmd() *cobra.Command {
	var noLogin bool // ignore ask for login
	var dockerContainerName, dockerImage string
	var options common.HelmOptions

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Run Testkube Docker Agent and connect to Testkube Pro environment",
		Aliases: []string{"install", "agent"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("WELCOME TO")
			ui.Logo()

			if strings.Contains(dockerImage, StableReleasePlaceholder) {
				latestVersion, err := common.GetLatestVersion()
				ui.ExitOnError("Getting latest version", err)
				dockerImage = strings.ReplaceAll(dockerImage, StableReleasePlaceholder, latestVersion)
			}

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
				ui.Warn("This will run Testkube Docker Agent latest version. This will take a few minutes.")
				ui.Warn("Please be sure you have Docker service running before continuing and can run containers in privileged mode!")
				ui.NL()

				dockerInfo, cliErr := common.RunDockerCommand([]string{"info"})
				if cliErr != nil {
					sendErrTelemetry(cmd, cfg, "docker_info", cliErr)
					common.HandleCLIError(cliErr)
				}
				ui.Alert("Current docker info:", dockerInfo)
				ui.NL()

				ok := ui.Confirm("Do you want to continue?")
				if !ok {
					ui.Errf("Testkube Docker Agent running cancelled")
					sendErrTelemetry(cmd, cfg, "user_cancel", errors.New("user cancelled agent running"))
					return
				}
			}

			var spinner *pterm.SpinnerPrinter
			if ui.IsVerbose() {
				ui.H2("Running Testkube Docker Agent")
			} else {
				spinner = ui.NewSpinner("Running Testkube Docker Agent")
			}

			if cliErr := common.DockerRunTestkubeAgent(options, cfg, dockerContainerName, dockerImage); cliErr != nil {
				if spinner != nil {
					spinner.Fail()
				}
				sendErrTelemetry(cmd, cfg, "docker_run", cliErr)
				common.HandleCLIError(cliErr)
			}

			if cliErr := common.StreamDockerLogs(dockerContainerName); cliErr != nil {
				if spinner != nil {
					spinner.Fail()
				}
				sendErrTelemetry(cmd, cfg, "docker_logs", cliErr)
				common.HandleCLIError(cliErr)
			}

			ui.NL()
			if spinner != nil {
				spinner.Success()
			} else {
				ui.Success("Testkube Docker Agent is up and running")
			}

			if noLogin {
				ui.Alert("Saving Testkube CLI Pro context, you need to authorize CLI through `testkube set context` later")
				cfg = common.PopulateCloudConfig(cfg, "", &dockerContainerName, &options)

				err = config.Save(cfg)
				ui.ExitOnError("saving config file", err)

				ui.Info(" Happy Testing! 🚀")
				ui.NL()
				return
			}

			ui.H2("Saving Testkube CLI Pro context")
			var token, refreshToken string
			if !common.IsUserLoggedIn(cfg, options) {
				token, refreshToken, err = common.LoginUser(options.Master.URIs.Auth, options.Master.CustomAuth, options.Master.CallbackPort)
				sendErrTelemetry(cmd, cfg, "login", err)
				ui.ExitOnError("user login", err)
			}
			err = common.PopulateLoginDataToContext(options.Master.OrgId, options.Master.EnvId, token, refreshToken, dockerContainerName, options, cfg)
			if err != nil {
				sendErrTelemetry(cmd, cfg, "setting_context", err)
				ui.ExitOnError("Setting Pro environment context", err)
			}
			ui.Info(" Happy Testing! 🚀")
			ui.NL()
		},
	}

	common.PopulateMasterFlags(cmd, &options, true)

	cmd.Flags().BoolVarP(&noLogin, "no-login", "", false, "Ignore login prompt, set existing token later by `testkube set context`")
	cmd.Flags().StringVar(&dockerContainerName, "docker-container", "testkube-agent", "Docker container name for Testkube Docker Agent")
	cmd.Flags().StringVar(&dockerImage, "docker-image", "kubeshop/testkube-agent:"+StableReleasePlaceholder, "Docker image for Testkube Docker Agent")

	return cmd
}

func sendErrTelemetry(cmd *cobra.Command, clientCfg config.Data, errType string, errorLogs error) {
	var errorStackTrace = fmt.Sprintf("%+v", errorLogs)
	if clientCfg.TelemetryEnabled {
		ui.Debug("collecting anonymous telemetry data, you can disable it by calling `kubectl testkube disable telemetry`")
		out, err := telemetry.SendCmdErrorEvent(cmd, common.Version, errType, errorStackTrace)
		if ui.Verbose && err != nil {
			ui.Err(err)
		}

		ui.Debug("telemetry send event response", out)
	}
}

func sendAttemptTelemetry(cmd *cobra.Command, clientCfg config.Data) {
	if clientCfg.TelemetryEnabled {
		ui.Debug("collecting anonymous telemetry data, you can disable it by calling `kubectl testkube disable telemetry`")
		out, err := telemetry.SendCmdAttemptEvent(cmd, common.Version)
		if ui.Verbose && err != nil {
			ui.Err(err)
		}
		ui.Debug("telemetry send event response", out)
	}
}
