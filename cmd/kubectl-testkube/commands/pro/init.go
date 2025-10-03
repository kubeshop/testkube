package pro

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	commonint "github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInitCmd() *cobra.Command {
	var export bool
	var noLogin bool // ignore ask for login
	var setOptions, argOptions map[string]string

	options := common.HelmOptions{
		NoMinio: true,
		NoMongo: true,
	}

	cmd := &cobra.Command{
		Use:     "agent",
		Short:   "Install Testkube Pro Agent and connect to Testkube Pro environment",
		Aliases: []string{"install", "agent", "init"},
		Run: func(cmd *cobra.Command, args []string) {
			if export {
				ui.Failf("export is unavailable for this profile")
				return
			}

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
				ui.Warn("This will install Testkube to the latest version. This may take a few minutes.")
				ui.Warn("Please be sure you're on valid kubectl context before continuing!")
				ui.NL()

				ui.NL()
				ui.H2("Running Kubectl command...")
				ui.NL()
				currentContext, cliErr := common.GetCurrentKubernetesContext()
				if cliErr != nil {
					sendErrTelemetry(cmd, cfg, "k8s_context", err)
					common.HandleCLIError(cliErr)
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

			spinner := ui.NewSpinner("Running Helm command...")
			options.SetOptions = setOptions
			options.ArgOptions = argOptions
			if cliErr := common.HelmUpgradeOrInstallTestkubeAgent(options, cfg, false); cliErr != nil {
				spinner.Fail()
				sendErrTelemetry(cmd, cfg, "helm_install", cliErr)
				common.HandleCLIError(cliErr)
			}

			spinner.Success()

			ui.NL()

			if noLogin {
				ui.Alert("Saving Testkube CLI Pro context, you need to authorize CLI through `testkube set context` later")
				cfg = common.PopulateCloudConfig(cfg, "", commonint.Ptr(""), &options)

				err = config.Save(cfg)
				ui.ExitOnError("saving config file", err)

				ui.Info(" Happy Testing! 🚀")
				ui.NL()
				return
			}

			ui.H2("Saving Testkube CLI Pro context")
			var token, refreshToken string
			if !common.IsUserLoggedIn(cfg, options) {
				ui.NL()
				ui.H2("Launching web browser...")
				ui.NL()
				token, refreshToken, err = common.LoginUser(options.Master.URIs.Auth, options.Master.CustomAuth, options.Master.CallbackPort)
				sendErrTelemetry(cmd, cfg, "login", err)
				ui.ExitOnError("user login", err)
			}
			err = common.PopulateLoginDataToContext(options.Master.OrgId, options.Master.EnvId, token, refreshToken, "", options, cfg)
			if err != nil {
				sendErrTelemetry(cmd, cfg, "setting_context", err)
				ui.ExitOnError("Setting Pro environment context", err)
			}
			ui.Info(" Happy Testing! 🚀")
			ui.NL()
		},
	}

	common.PopulateHelmFlags(cmd, &options)
	common.PopulateMasterFlags(cmd, &options, false)

	cmd.Flags().BoolVarP(&noLogin, "no-login", "", false, "Ignore login prompt, set existing token later by `testkube set context`")
	cmd.Flags().BoolVarP(&export, "export", "", false, "Export the values.yaml")
	cmd.Flags().BoolVar(&options.MultiNamespace, "multi-namespace", false, "multi namespace mode")
	cmd.Flags().BoolVar(&options.NoOperator, "no-operator", false, "should operator be installed (for more instances in multi namespace mode it should be set to true)")
	cmd.Flags().StringToStringVarP(&setOptions, "helm-set", "", nil, "helm set option in form of key=value")
	cmd.Flags().StringToStringVarP(&argOptions, "helm-arg", "", nil, "helm arg option in form of key=value")

	return cmd
}

func sendErrTelemetry(cmd *cobra.Command, clientCfg config.Data, errType string, errorLogs error) {
	var errorStackTrace string = fmt.Sprintf("%+v", errorLogs)
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
