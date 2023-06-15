package context

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewSetContextCmd() *cobra.Command {
	var (
		org, env, apiKey string
		agentKey         string
		kubeconfig       bool
		namespace        string
		rootDomain       string
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
				if org == "" && env == "" && apiKey == "" && agentKey == "" && rootDomain == "" {
					ui.Errf("Please provide at least one of the following flags: --org, --env, --api-key, --cloud-root-domain, --cloud-agent-key")
				}

				if org != "" {
					cfg.CloudContext.OrganizationId = org
					// reset env when the org is changed
					if env == "" {
						cfg.CloudContext.EnvironmentId = ""
					}
				}
				if env != "" {
					cfg.CloudContext.EnvironmentId = env
				}
				if apiKey != "" {
					cfg.CloudContext.ApiKey = apiKey
				}

				// set uris based on root domain
				uris := common.NewCloudUris(rootDomain)
				cfg.CloudContext.ApiUri = uris.Api
				cfg.CloudContext.UiUri = uris.Ui
				cfg.CloudContext.AgentUri = uris.Agent

				orgClient := cloudclient.NewOrganizationsClient(rootDomain, cfg.CloudContext.ApiKey)
				ui.ExitOnError("getting client", err)
				org, err := orgClient.Get(cfg.CloudContext.OrganizationId)
				ui.ExitOnError("getting organization", err)

				envsClient := cloudclient.NewEnvironmentsClient(rootDomain, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId)
				env, err := envsClient.Get(cfg.CloudContext.EnvironmentId)
				ui.ExitOnError("getting environment", err)

				cfg.CloudContext.OrganizationName = org.Name
				cfg.CloudContext.EnvironmentName = env.Name

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
	cmd.Flags().StringVarP(&rootDomain, "cloud-root-domain", "", "testkube.io", "defaults to testkube.io, usually you don't need to change it")

	return cmd
}
