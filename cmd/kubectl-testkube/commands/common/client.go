package common

import (
	"strconv"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

// GetClient returns api client
func GetClient(cmd *cobra.Command) (client.Client, string) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	apiURI := cmd.Flag("api-uri").Value.String()
	oauthEnabled, err := strconv.ParseBool(cmd.Flag("oauth-enabled").Value.String())
	ui.ExitOnError("parsing flag value", err)

	options := client.Options{
		Namespace:      namespace,
		APIURI:         apiURI,
		OAuthLocalPort: oauth.LocalPort,
	}

	if oauthEnabled {
		cfg, err := config.Load()
		ui.ExitOnError("loading config file", err)

		options.Config = &cfg.OAuth2Data.Config
		options.Token = cfg.OAuth2Data.Token
		if options.Token == nil {
			ui.ExitOnError("oauth token is empty, please configure your oauth settings first")
		}
	}

	client, err := client.GetClient(client.ClientType(clientType), options)
	ui.ExitOnError("setting up client type", err)

	return client, namespace
}
