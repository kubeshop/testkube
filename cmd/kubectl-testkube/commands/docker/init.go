package docker

import (
	"errors"
	"fmt"
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

	configFileHint = "Check is the Testkube config file (~/.testkube/config.json) accessible and has right permissions"
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

			cfg, err := config.Load()
			if err != nil {
				// The config is not loaded yet, so this failure can't be reported to telemetry.
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					configFileHint,
					err,
				))
			}
			ui.NL()
			skipTLS := common.SyncSkipTLSFromFlags(cmd, &cfg)

			common.ProcessMasterFlags(cmd, &options, &cfg)

			sendAttemptTelemetry(cmd, cfg)

			if strings.Contains(dockerImage, StableReleasePlaceholder) {
				latestVersion, err := common.GetLatestVersion()
				if err != nil {
					exitOnCLIError(cmd, cfg, "latest_version", common.NewCLIError(
						common.TKErrLatestVersionFetchFailed,
						"Error getting the latest Testkube version",
						"Check your internet connection and access to github.com, or pin the version with the '--docker-image' flag",
						err,
					))
				}
				dockerImage = strings.ReplaceAll(dockerImage, StableReleasePlaceholder, latestVersion)
			}

			if !options.NoConfirm {
				ui.Warn("This will run Testkube Docker Agent latest version. This will take a few minutes.")
				ui.Warn("Please be sure you have Docker service running before continuing and can run containers in privileged mode!")
				ui.NL()

				dockerInfo, cliErr := common.RunDockerCommand([]string{"info"})
				exitOnCLIError(cmd, cfg, "docker_info", cliErr)
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
				exitOnCLIError(cmd, cfg, "docker_run", cliErr)
			}

			if cliErr := common.StreamDockerLogs(dockerContainerName); cliErr != nil {
				if spinner != nil {
					spinner.Fail()
				}
				exitOnCLIError(cmd, cfg, "docker_logs", cliErr)
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

				if err = config.Save(cfg); err != nil {
					exitOnCLIError(cmd, cfg, "saving_config", common.NewCLIError(
						common.TKErrConfigSaveFailed,
						"Error saving testkube config file",
						configFileHint,
						err,
					))
				}

				ui.Info(" Happy Testing! 🚀")
				ui.NL()
				return
			}

			ui.H2("Saving Testkube CLI Pro context")
			var token, refreshToken string
			// Preserve the existing session's token type; fall back to OIDC
			// for brand-new contexts with nothing stored yet.
			tokenType := cfg.CloudContext.TokenType
			if tokenType == "" {
				tokenType = config.TokenTypeOIDC
			}
			if !common.IsUserLoggedIn(cfg, options) {
				tokenType, token, refreshToken, err = common.LoginUser(options.Master.URIs.Auth, options.Master.URIs.Api, options.Master.CustomAuth, options.Master.CallbackPort, skipTLS)
				if err != nil {
					exitOnCLIError(cmd, cfg, "login", common.NewCLIError(
						common.TKErrLoginFailed,
						"Error logging in to Testkube Pro",
						"Check is the browser able to reach the Testkube Pro auth endpoint, or skip the login with the '--no-login' flag and set the context later by `testkube set context`",
						err,
					))
				}
			}
			// A user who was already logged in has no fresh token here, so fall
			// back to the one stored in the context to look org/env up with.
			lookupToken := token
			if lookupToken == "" {
				lookupToken = cfg.CloudContext.ApiKey
			}
			// The organization has to resolve first: the environment lookup is
			// scoped to it.
			orgID, err := common.ResolveOrgOrPrompt(options.Master.URIs.Api, lookupToken, options.Master, skipTLS)
			if err != nil {
				exitOnCLIError(cmd, cfg, "setting_context", common.NewCLIError(
					common.TKErrOrgResolutionFailed,
					"Error resolving Testkube Pro organization",
					"Check does your account have access to the organization, or select it explicitly with the '--org-id' flag",
					err,
				))
			}

			envID, err := common.ResolveEnvOrPrompt(options.Master.URIs.Api, lookupToken, orgID, options.Master, skipTLS)
			if err != nil {
				exitOnCLIError(cmd, cfg, "setting_context", common.NewCLIError(
					common.TKErrEnvResolutionFailed,
					"Error resolving Testkube Pro environment",
					"Check does the environment exist in the selected organization, or select it explicitly with the '--env-id' flag",
					err,
				))
			}

			if err = common.PopulateLoginDataToContext(orgID, envID, tokenType, token, refreshToken, dockerContainerName, options, cfg); err != nil {
				exitOnCLIError(cmd, cfg, "setting_context", common.NewCLIError(
					common.TKErrContextSaveFailed,
					"Error setting Testkube Pro environment context",
					configFileHint,
					err,
				))
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

// exitOnCLIError reports the failure to telemetry and then prints the rich
// error details and terminates the command. It is a no-op for a nil error.
func exitOnCLIError(cmd *cobra.Command, clientCfg config.Data, errType string, cliErr *common.CLIError) {
	if cliErr == nil {
		return
	}

	sendErrTelemetry(cmd, clientCfg, errType, cliErr)
	common.HandleCLIError(cliErr)
}

func sendErrTelemetry(cmd *cobra.Command, clientCfg config.Data, errType string, errorLogs error) {
	if !clientCfg.TelemetryEnabled {
		return
	}

	var errorStackTrace = fmt.Sprintf("%+v", errorLogs)
	// Carry the TKERR code over when we know it, so the failure is groupable.
	var errCode string
	var cliErr *common.CLIError
	if errors.As(errorLogs, &cliErr) {
		errCode = string(cliErr.Code)
		errorStackTrace = cliErr.StackTrace
	}

	ui.Debug("collecting anonymous telemetry data, you can disable it by calling `testkube disable telemetry`")
	out, err := telemetry.SendCmdErrorEventWithLicense(cmd, common.Version, errType, errorStackTrace, "", "", errCode)
	if ui.Verbose && err != nil {
		ui.Err(err)
	}

	ui.Debug("telemetry send event response", out)
}

func sendAttemptTelemetry(cmd *cobra.Command, clientCfg config.Data) {
	if clientCfg.TelemetryEnabled {
		ui.Debug("collecting anonymous telemetry data, you can disable it by calling `testkube disable telemetry`")
		out, err := telemetry.SendCmdAttemptEvent(cmd, common.Version, common.TelemetryUserID(cmd, &clientCfg))
		if ui.Verbose && err != nil {
			ui.Err(err)
		}
		ui.Debug("telemetry send event response", out)
	}
}
