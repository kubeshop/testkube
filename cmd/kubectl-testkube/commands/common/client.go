package common

import (
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

	var token *oauth2.Token
	var oauthCfg *oauth2.Config
	if oauthEnabled {
		cfg, err := config.Load()
		ui.ExitOnError("loading config file", err)

		oauthCfg = &cfg.OAuth2Data.Config
		token = cfg.OAuth2Data.Token
		if token == nil {
			ui.ExitOnError("oauth token is empty")
		}
	}

	client, err := client.GetClient(client.ClientType(clientType), namespace, apiURI, token, oauthCfg)
	ui.ExitOnError("setting up client type", err)

	return client, namespace
}
