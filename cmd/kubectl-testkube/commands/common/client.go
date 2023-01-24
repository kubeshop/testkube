package common

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
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
		ApiUri:    apiURI,
	}

	cfg, err := config.Load()
	ui.ExitOnError("loading config file", err)

	switch cfg.ContextType {
	case config.ContextTypeKubeconfig:
		if oauthEnabled {
			options.Provider = cfg.OAuth2Data.Provider
			options.ClientID = cfg.OAuth2Data.ClientID
			options.ClientSecret = cfg.OAuth2Data.ClientSecret
			options.Scopes = cfg.OAuth2Data.Scopes
			options.Token = cfg.OAuth2Data.Token

			if os.Getenv("TESTKUBE_OAUTH_ACCESS_TOKEN") != "" {
				options.Token = &oauth2.Token{
					AccessToken: os.Getenv("TESTKUBE_OAUTH_ACCESS_TOKEN"),
				}
			}

			if options.Token == nil {
				ui.ExitOnError("oauth token is empty, please configure your oauth settings first")
			}
		}
	case config.ContextTypeCloud:
		clientType = string(client.ClientCloud)
		options.CloudApiPathPrefix = fmt.Sprintf("/organizations/%s/environments/%s/agent", cfg.CloudContext.Organization, cfg.CloudContext.Environment)
		options.CloudApiKey = cfg.CloudContext.ApiKey
		options.CloudEnvironment = cfg.CloudContext.Environment
		options.CloudOrganization = cfg.CloudContext.Organization
		options.ApiUri = cfg.CloudContext.ApiUri
	}

	c, err := client.GetClient(client.ClientType(clientType), options)
	ui.ExitOnError("setting up client type", err)

	return c, namespace
}
