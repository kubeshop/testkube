package pro

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/spf13/cobra"
)

func NewLoginCmd() *cobra.Command {
	var opts common.HelmOptions

	cmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"d"},
		Short:   "Login to Testkube Pro",
		Run: func(cmd *cobra.Command, args []string) {
			opts.CloudUris = common.NewCloudUris(opts.CloudRootDomain)
			token, refreshToken, err := common.LoginUser(opts.CloudUris.Auth)
			ui.ExitOnError("getting token", err)

			orgID := opts.CloudOrgId
			envID := opts.CloudEnvId

			if orgID == "" {
				orgID, _, err = uiGetOrganizationId(opts.CloudRootDomain, token)
				ui.ExitOnError("getting organization", err)
			}
			if envID == "" {
				envID, _, err = uiGetEnvironmentID(opts.CloudRootDomain, token, orgID)
				ui.ExitOnError("getting environment", err)
			}
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			err = common.PopulateLoginDataToContext(orgID, envID, token, refreshToken, opts, cfg)
			ui.ExitOnError("saving config file", err)

			ui.Success("Your config was updated with new values")
			ui.NL()
			common.UiPrintContext(cfg)
		},
	}

	common.PopulateProUriFlags(cmd, &opts)

	cmd.Flags().StringVar(&opts.CloudOrgId, "org-id", "", "Testkube Pro organization id")
	cmd.Flags().StringVar(&opts.CloudEnvId, "env-id", "", "Testkube Pro environment id")

	return cmd
}
