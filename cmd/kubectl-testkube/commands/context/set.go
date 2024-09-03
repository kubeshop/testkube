package context

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewSetContextCmd() *cobra.Command {
	var (
		org, env, apiKey string
		kubeconfig       bool
		namespace        string
		opts             common.HelmOptions
	)

	cmd := &cobra.Command{
		Use:   "context <value>",
		Short: "Set context data for Testkube Pro",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)
			common.ProcessMasterFlags(cmd, &opts, &cfg)

			if cmd.Flags().Changed("org") {
				opts.Master.OrgId = org
			}

			if cmd.Flags().Changed("env") {
				opts.Master.EnvId = env
			}

			if kubeconfig {
				cfg.ContextType = config.ContextTypeKubeconfig
			} else {
				cfg.ContextType = config.ContextTypeCloud
			}

			switch cfg.ContextType {
			case config.ContextTypeCloud:
				if opts.Master.OrgId == "" && opts.Master.EnvId == "" && apiKey == "" && opts.Master.RootDomain == "" {
					ui.Errf("Please provide at least one of the following flags: --org-id, --env-id, --api-key, --root-domain")
				}

				cfg = common.PopulateCloudConfig(cfg, apiKey, &opts)

				if cfg.CloudContext.ApiKey != "" {
					var err error
					cfg, err = common.PopulateOrgAndEnvNames(cfg, opts.Master.OrgId, opts.Master.EnvId, opts.Master.URIs.Api)
					if err != nil {
						ui.Errf("Error populating org and env names: %s", err)
					}
				} else {
					ui.Warn("No API key provided, you need to login to Testkube Cloud")
				}

			case config.ContextTypeKubeconfig:
				// kubeconfig special use cases

			default:
				ui.Errf("Unknown context type: %s", cfg.ContextType)
			}

			if namespace != "" {
				cfg.Namespace = namespace
			}

			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)

			if err = validator.ValidateCloudContext(cfg); err != nil {
				common.UiCloudContextValidationError(err)
			}

			common.UiPrintContext(cfg)
			ui.NL()
			ui.Success("Your config was updated with new values")
		},
	}

	cmd.Flags().BoolVarP(&kubeconfig, "kubeconfig", "", false, "reset context mode for CLI to default kubeconfig based")
	cmd.Flags().StringVarP(&org, "org", "o", "", "Testkube Pro Organization ID")
	cmd.Flags().MarkDeprecated("org", "use --org-id instead")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Testkube Pro Environment ID")
	cmd.Flags().MarkDeprecated("env", "use --env-id instead")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Testkube namespace to use for CLI commands")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API Key for Testkube Pro")

	// allow to override default values of all URIs
	cmd.Flags().String("api-uri-override", "", "api uri override")
	cmd.Flags().String("ui-uri-override", "", "ui uri override")
	cmd.Flags().String("agent-uri-override", "", "agnet uri override")
	cmd.Flags().String("logs-uri-override", "", "logs service uri override")

	common.PopulateMasterFlags(cmd, &opts)
	return cmd
}
