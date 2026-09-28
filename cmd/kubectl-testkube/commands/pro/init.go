package pro

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	commonint "github.com/kubeshop/testkube/internal/common"
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
		Use:     "runner",
		Short:   "Install Testkube Pro Runner and connect to Testkube Pro environment",
		Aliases: []string{"install", "agent", "init"},
		Run: func(cmd *cobra.Command, args []string) {
			if export {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrInvalidRuntimeParameter,
					"Export is unavailable for this profile",
					"Drop the '--export' flag, it is only supported when installing the standalone runner",
					errors.New("export is unavailable for this profile"),
				))
			}

			ui.Info("WELCOME TO")
			ui.Logo()

			cfg, err := config.Load()
			if err != nil {
				// The config is not loaded yet, so this failure can't be reported to telemetry.
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					common.ConfigFileHint,
					err,
				))
			}
			ui.NL()
			skipTLS := common.SyncSkipTLSFromFlags(cmd, &cfg)

			common.ProcessMasterFlags(cmd, &options, &cfg)
			common.ShowOperatorDeprecationWarning("Testkube Runner", options.NoCRDs)

			common.SendAttemptTelemetry(cmd, cfg)

			if !options.NoConfirm {
				ui.Warn("This will install Testkube to the latest version. This may take a few minutes.")
				ui.Warn("Please be sure you're on valid kubectl context before continuing!")
				ui.NL()

				ui.NL()
				ui.H2("Running Kubectl command...")
				ui.NL()
				currentContext, cliErr := common.GetCurrentKubernetesContext()
				common.ExitOnCLIError(cmd, cfg, "k8s_context", cliErr)
				ui.Alert("Current kubectl context:", currentContext)
				ui.NL()

				ok := ui.Confirm("Do you want to continue?")
				if !ok {
					ui.Errf("Testkube installation cancelled")
					common.SendErrTelemetry(cmd, cfg, "user_cancel", errors.New("user cancelled installation"))
					return
				}
			}

			spinner := ui.NewSpinner("Running Helm command...")
			options.SetOptions = setOptions
			options.ArgOptions = argOptions
			if cliErr := common.HelmUpgradeOrInstallTestkubeAgent(options, cfg, false); cliErr != nil {
				spinner.Fail()
				common.ExitOnCLIError(cmd, cfg, "helm_install", cliErr)
			}

			spinner.Success()

			ui.NL()

			if noLogin {
				ui.Alert("Saving Testkube CLI Pro context, you need to authorize CLI through `testkube set context` later")
				cfg = common.PopulateCloudConfig(cfg, "", commonint.Ptr(""), &options)

				if err = config.Save(cfg); err != nil {
					common.ExitOnCLIError(cmd, cfg, "saving_config", common.NewCLIError(
						common.TKErrConfigSaveFailed,
						"Error saving testkube config file",
						common.ConfigFileHint,
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
				ui.NL()
				ui.H2("Launching web browser...")
				ui.NL()
				tokenType, token, refreshToken, err = common.LoginUser(options.Master.URIs.Auth, options.Master.URIs.Api, options.Master.CustomAuth, options.Master.CallbackPort, skipTLS)
				if err != nil {
					common.ExitOnCLIError(cmd, cfg, "login", common.NewCLIError(
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
				common.ExitOnCLIError(cmd, cfg, "setting_context", common.NewCLIError(
					common.TKErrOrgResolutionFailed,
					"Error resolving Testkube Pro organization",
					"Check does your account have access to the organization, or select it explicitly with the '--org-id' flag",
					err,
				))
			}

			envID, err := common.ResolveEnvOrPrompt(options.Master.URIs.Api, lookupToken, orgID, options.Master, skipTLS)
			if err != nil {
				common.ExitOnCLIError(cmd, cfg, "setting_context", common.NewCLIError(
					common.TKErrEnvResolutionFailed,
					"Error resolving Testkube Pro environment",
					"Check does the environment exist in the selected organization, or select it explicitly with the '--env-id' flag",
					err,
				))
			}

			if err = common.PopulateLoginDataToContext(orgID, envID, tokenType, token, refreshToken, "", options, cfg); err != nil {
				common.ExitOnCLIError(cmd, cfg, "setting_context", common.NewCLIError(
					common.TKErrContextSaveFailed,
					"Error setting Testkube Pro environment context",
					common.ConfigFileHint,
					err,
				))
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
	cmd.Flags().BoolVar(&options.NoCRDs, "no-crds", false, "Skip installing CRDs, useful when you have them already installed in the cluster")
	cmd.Flags().StringToStringVarP(&setOptions, "helm-set", "", nil, "helm set option in form of key=value")
	cmd.Flags().StringToStringVarP(&argOptions, "helm-arg", "", nil, "helm arg option in form of key=value")

	return cmd
}
