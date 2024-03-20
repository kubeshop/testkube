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
					ui.Errf("Please provide at least one of the following flags: --org, --env, --api-key, --root-domain")
				}

				cfg = common.PopulateCloudConfig(cfg, apiKey, &opts)

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

			ui.Success("Your config was updated with new values")
			ui.NL()
			common.UiPrintContext(cfg)

		},
	}

	cmd.Flags().BoolVarP(&kubeconfig, "kubeconfig", "", false, "reset context mode for CLI to default kubeconfig based")
	cmd.Flags().StringVarP(&org, "org", "o", "", "Testkube Pro Organization ID")
	cmd.Flags().MarkDeprecated("org", "use --org-id instead")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Testkube Pro Environment ID")
	cmd.Flags().MarkDeprecated("env", "use --env-id instead")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Testkube namespace to use for CLI commands")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API Key for Testkube Pro")

	common.PopulateMasterFlags(cmd, &opts)
	return cmd
}
