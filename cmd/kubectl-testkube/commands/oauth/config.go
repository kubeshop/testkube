package oauth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/oauth"
	poauth "github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewConfigureOAuthCmd is oauth config config cmd
func NewConfigureOAuthCmd() *cobra.Command {
	var (
		providerType string
		clientID     string
		clientSecret string
		scopes       []string
	)

	cmd := &cobra.Command{
		Use:   "oauth <value>",
		Short: "Set oauth credentials for api uri in testkube client",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please pass valid api uri value")
			}

			values := map[string]string{
				"client id":     clientID,
				"client secret": clientSecret,
			}

			for key, value := range values {
				if value == "" {
					return fmt.Errorf("please pass valid %s value", key)
				}
			}

			provider := poauth.NewProvider(clientID, clientSecret, scopes)
			if _, err := provider.GetValidator(poauth.ProviderType(providerType)); err != nil {
				return err
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.APIURI = args[0]
			cfg.OAuth2Data.Provider = poauth.ProviderType(providerType)
			cfg.OAuth2Data.ClientID = clientID
			cfg.OAuth2Data.ClientSecret = clientSecret
			cfg.OAuth2Data.Scopes = scopes

			provider := poauth.NewProvider(clientID, clientSecret, scopes)
			client, err := provider.AuthenticateUser(poauth.ProviderType(providerType))
			ui.ExitOnError("authenticating user", err)

			cfg.OAuth2Data.Token = client.Token
			cfg.EnableOAuth()
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New api uri set to", cfg.APIURI)
			ui.Success("New oauth token", cfg.OAuth2Data.Token.AccessToken)
		},
	}

	cmd.Flags().StringVar(&providerType, "provider", string(oauth.GithubProviderType), "authentication provider, currently available: github")
	cmd.Flags().StringVar(&clientID, "client-id", "", "client id for authentication provider")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "client secret for authentication provider")
	cmd.Flags().StringArrayVar(&scopes, "scope", nil, "scope for authentication provider")

	return cmd
}
