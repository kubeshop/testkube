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
		org, env, apiKey                 string
		kubeconfig, insecureClient       bool
		namespace                        string
		rootDomain                       string
		uiPrefix, apiPrefix, agentPrefix string
	)

	cmd := &cobra.Command{
		Use:   "context <value>",
		Short: "Set context data for Testkube Pro",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if kubeconfig {
				cfg.ContextType = config.ContextTypeKubeconfig
			} else {
				cfg.ContextType = config.ContextTypeCloud
			}

			switch cfg.ContextType {
			case config.ContextTypeCloud:
				if org == "" && env == "" && apiKey == "" && rootDomain == "" {
					ui.Errf("Please provide at least one of the following flags: --org, --env, --api-key, --cloud-root-domain")
				}

				cfg = common.PopulateCloudConfig(cfg, apiKey, org, env, rootDomain, apiPrefix, uiPrefix, agentPrefix, insecureClient)

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
	cmd.Flags().StringVarP(&org, "org", "o", "", "Testkube Cloud Organization ID")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Testkube Cloud Environment ID")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Testkube namespace to use for CLI commands")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API Key for Testkube Cloud")

	cmd.Flags().BoolVar(&insecureClient, "cloud-insecure", false, "should client connect in insecure mode (will use http instead of https)")
	cmd.Flags().StringVar(&agentPrefix, "cloud-agent-prefix", "agent", "defaults to 'agent', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&apiPrefix, "cloud-api-prefix", "api", "defaults to 'api', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&uiPrefix, "cloud-ui-prefix", "ui", "defaults to 'ui', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&rootDomain, "cloud-root-domain", "testkube.io", "defaults to testkube.io, usually don't need to be changed [required for custom cloud mode]")

	return cmd
}
