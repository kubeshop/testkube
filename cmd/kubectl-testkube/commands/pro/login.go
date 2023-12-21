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
		Aliases: []string{"l"},
		Short:   "Login to Testkube Pro",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			common.ProcessMasterFlags(cmd, &opts, &cfg)

			token, refreshToken, err := common.LoginUser(opts.Master.URIs.Auth)
			ui.ExitOnError("getting token", err)

			orgID := opts.Master.OrgId
			envID := opts.Master.EnvId

			if orgID == "" {
				orgID, _, err = common.UiGetOrganizationId(opts.Master.URIs.Api, token)
				ui.ExitOnError("getting organization", err)
			}
			if envID == "" {
				envID, _, err = common.UiGetEnvironmentID(opts.Master.URIs.Api, token, orgID)
				ui.ExitOnError("getting environment", err)
			}

			err = common.PopulateLoginDataToContext(orgID, envID, token, refreshToken, opts, cfg)
			ui.ExitOnError("saving config file", err)

			ui.Success("Your config was updated with new values")
			ui.NL()
			common.UiPrintContext(cfg)
		},
	}

	common.PopulateMasterFlags(cmd, &opts)

	return cmd
}
