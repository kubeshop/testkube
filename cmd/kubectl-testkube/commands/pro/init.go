package pro

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInitCmd() *cobra.Command {
	var export bool
	options := common.HelmOptions{
		NoMinio: true,
		NoMongo: true,
	}

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Install Testkube Pro Agent and connect to Testkube Pro environment",
		Aliases: []string{"install", "agent"},
		Run: func(cmd *cobra.Command, args []string) {
			if export {
				ui.Failf("export is unavailable for this profile")
				return
			}

			ui.Info("WELCOME TO")
			ui.Logo()

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)
			ui.NL()

			common.ProcessMasterFlags(cmd, &options, &cfg)

			sendAttemptTelemetry(cmd, cfg)

			if !options.NoConfirm {
				ui.Warn("This will install Testkube to the latest version. This may take a few minutes.")
				ui.Warn("Please be sure you're on valid kubectl context before continuing!")
				ui.NL()

				currentContext, err := common.GetCurrentKubernetesContext()

				if err != nil {
					sendErrTelemetry(cmd, cfg, "k8s_context", err)
					ui.ExitOnError("getting current context", err)
				}
				ui.Alert("Current kubectl context:", currentContext)
				ui.NL()

				ok := ui.Confirm("Do you want to continue?")
				if !ok {
					ui.Errf("Testkube installation cancelled")
					sendErrTelemetry(cmd, cfg, "user_cancel", err)
					return
				}
			}

			spinner := ui.NewSpinner("Installing Testkube")
			err = common.HelmUpgradeOrInstallTestkubeCloud(options, cfg, false)
			if err != nil {
				sendErrTelemetry(cmd, cfg, "helm_install", err)
				ui.ExitOnError("Installing Testkube", err)
			}
			spinner.Success()

			ui.NL()

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

	common.PopulateHelmFlags(cmd, &options)
	common.PopulateMasterFlags(cmd, &options)

	cmd.Flags().BoolVarP(&export, "export", "", false, "Export the values.yaml")
	cmd.Flags().BoolVar(&options.MultiNamespace, "multi-namespace", false, "multi namespace mode")
	cmd.Flags().BoolVar(&options.NoOperator, "no-operator", false, "should operator be installed (for more instances in multi namespace mode it should be set to true)")

	return cmd
}

func sendErrTelemetry(cmd *cobra.Command, clientCfg config.Data, errType string, errorLogs error) {
	var errorStackTrace string
	errorStackTrace = fmt.Sprintf("%+v", errorLogs)
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
