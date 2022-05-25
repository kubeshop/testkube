package common

import (
	"os"
	"strconv"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// GetClient returns api client
func GetClient(cmd *cobra.Command) (client.Client, string) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	apiURI := cmd.Flag("api-uri").Value.String()
	oauthEnabled, err := strconv.ParseBool(cmd.Flag("oauth-enabled").Value.String())
	ui.ExitOnError("parsing flag value", err)

	options := client.Options{
		Namespace: namespace,
		APIURI:    apiURI,
	}

	if oauthEnabled {
		cfg, err := config.Load()
		ui.ExitOnError("loading config file", err)

		options.Provider = cfg.OAuth2Data.Provider
		options.ClientID = cfg.OAuth2Data.ClientID
		options.ClientSecret = cfg.OAuth2Data.ClientSecret
		options.Scopes = cfg.OAuth2Data.Scopes
		options.Token = cfg.OAuth2Data.Token
		if options.Token == nil && os.Getenv("TESTKUBE_OAUTH_ACCESS_TOKEN") != "" {
			options.Token = &oauth2.Token{
				AccessToken: os.Getenv("TESTKUBE_OAUTH_ACCESS_TOKEN"),
			}
		}

		if options.Token == nil {
			ui.ExitOnError("oauth token is empty, please configure your oauth settings first")
		}
	}

	client, err := client.GetClient(client.ClientType(clientType), options)
	ui.ExitOnError("setting up client type", err)

	return client, namespace
}
