package cloud

import (
	"context"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/cloudlogin"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

func NewLoginCmd() *cobra.Command {
	var opts common.HelmOptions

	cmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"d"},
		Short:   "Login to Testkube Cloud",
		Run: func(cmd *cobra.Command, args []string) {
			opts.CloudUris = common.NewCloudUris(opts.CloudRootDomain)
			authUrl, tokenChan, err := cloudlogin.CloudLogin(context.Background(), opts.CloudUris.Auth)
			ui.ExitOnError("cloud login", err)

			ui.H1("Login")
			ui.Paragraph("Your browser should open automatically. If not, please open this link in your browser:")
			ui.Link(authUrl)
			ui.Paragraph("(just login and get back to your terminal)")
			ui.Paragraph("")

			if ok := ui.Confirm("Continue"); !ok {
				return
			}

			// open browser with login page and redirect to localhost
			open.Run(authUrl)

			token, refreshToken, err := uiGetToken(tokenChan)
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

	cmd.Flags().StringVar(&opts.CloudRootDomain, "cloud-root-domain", "testkube.io", "defaults to testkube.io, usually don't need to be changed [required for cloud mode]")

	cmd.Flags().StringVar(&opts.CloudOrgId, "org-id", "", "Testkube Cloud organization id")
	cmd.Flags().StringVar(&opts.CloudEnvId, "env-id", "", "Testkube Cloud environment id")

	return cmd
}
