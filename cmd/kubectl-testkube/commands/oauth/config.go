package oauth

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	poauth "github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/github"
)

// NewConfigureOAuthCmd is oauth config config cmd
func NewConfigureOAuthCmd(port int) *cobra.Command {
	var (
		authURI      string
		tokenURI     string
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
				"auth uri":      authURI,
				"token uri":     tokenURI,
				"client id":     clientID,
				"client secret": clientSecret,
			}

			for key, value := range values {
				if value == "" {
					return fmt.Errorf("please pass valid %s value", key)
				}
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.APIURI = args[0]
			cfg.OAuth2Data.Config.Endpoint.AuthURL = authURI
			cfg.OAuth2Data.Config.Endpoint.TokenURL = tokenURI
			cfg.OAuth2Data.Config.ClientID = clientID
			cfg.OAuth2Data.Config.ClientSecret = clientSecret
			cfg.OAuth2Data.Config.Scopes = scopes

			provider := poauth.NewProvider(&cfg.OAuth2Data.Config, port)
			client, err := provider.AuthenticateUser(nil)
			ui.ExitOnError("authenticating user", err)

			cfg.OAuth2Data.Token = client.Token
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New api uri set to", cfg.APIURI)
			ui.Success("New oauth token", cfg.OAuth2Data.Token.AccessToken)
		},
	}

	cmd.Flags().StringVar(&authURI, "auth-uri", github.Endpoint.AuthURL, "auth uri for authentication provider (github is a default provider)")
	cmd.Flags().StringVar(&tokenURI, "token-uri", github.Endpoint.TokenURL, "token uri for authentication provider (github is a default provider)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "client id for authentication provider")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "client secret for authentication provider")
	cmd.Flags().StringArrayVar(&scopes, "scope", nil, "scope for authentication provider")

	return cmd
}
