package context

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewSetContextCmd() *cobra.Command {
	var org, env, apiKey string

	cmd := &cobra.Command{
		Use:   "context <value>",
		Short: "Set namespace for testkube client",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if org == "" && env == "" && apiKey == "" {
				ui.Errf("Please provide at least one of the following flags: --org, --env, --api-key")
			}

			if org != "" {
				cfg.CloudContext.Organization = org
			}
			if env != "" {
				cfg.CloudContext.Environment = env
			}
			if org != "" {
				cfg.CloudContext.ApiKey = apiKey
			}

			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)

			ui.Success("Your config was updated with new values")
			ui.NL()
			uiPrintCloudContext(cfg.CloudContext)
		},
	}

	cmd.Flags().StringVarP(&org, "org", "o", "", "Testkube Cloud organization ID")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Testkube Cloud environment ID")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API Key for Testkube Cloud")

	return cmd
}
