package context

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewSetContextCmd() *cobra.Command {
	var (
		org, env, apiKey, apiUri string
		kubeconfig               bool
	)

	cmd := &cobra.Command{
		Use:   "context <value>",
		Short: "Set context data for Testkube Cloud",
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
				if org == "" && env == "" && apiKey == "" {
					ui.Errf("Please provide at least one of the following flags: --org, --env, --api-key")
				}

				if org != "" {
					cfg.CloudContext.Organization = org
					// reset env when the org is changed
					if env == "" {
						cfg.CloudContext.Environment = ""
					}
				}
				if env != "" {
					cfg.CloudContext.Environment = env
				}
				if apiKey != "" {
					cfg.CloudContext.ApiKey = apiKey
					cfg.CloudContext.ApiUri = apiUri
				}
			case config.ContextTypeKubeconfig:
				// kubecofnig special use cases

			default:
				ui.Errf("Unknown context type: %s", cfg.ContextType)
			}

			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)

			ui.Success("Your config was updated with new values")
			ui.NL()
			uiPrintCloudContext(cfg.CloudContext)
		},
	}

	cmd.Flags().BoolVarP(&kubeconfig, "kubeconfig", "", false, "Default kubeconfig based mode for CLI")
	cmd.Flags().StringVarP(&org, "org", "o", "", "Testkube Cloud organization ID")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Testkube Cloud environment ID")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API Key for Testkube Cloud")
	cmd.Flags().StringVarP(&apiUri, "cloud-api-uri", "", "", "https://api.testkube.io")

	return cmd
}
