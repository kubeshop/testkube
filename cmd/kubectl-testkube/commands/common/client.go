package common

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
)

// GetClient returns api client
func GetClient(cmd *cobra.Command) (client.Client, string, error) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	apiURI := cmd.Flag("api-uri").Value.String()
	oauthEnabled, err := strconv.ParseBool(cmd.Flag("oauth-enabled").Value.String())
	if err != nil {
		return nil, "", fmt.Errorf("parsing flag value %w", err)
	}

	options := client.Options{
		Namespace: namespace,
		ApiUri:    apiURI,
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, "", fmt.Errorf("loading config file %w", err)
	}

	// set kubeconfig as default config type
	if cfg.ContextType == "" {
		cfg.ContextType = config.ContextTypeKubeconfig
	}

	if cfg.APIServerName == "" {
		cfg.APIServerName = config.APIServerName
	}

	if cfg.APIServerPort == 0 {
		cfg.APIServerPort = config.APIServerPort
	}

	options.APIServerName = cfg.APIServerName
	options.APIServerPort = cfg.APIServerPort

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
				return nil, "", errors.New("oauth token is empty, please configure your oauth settings first")
			}
		}
	case config.ContextTypeCloud:
		clientType = string(client.ClientCloud)
		options.CloudApiPathPrefix = fmt.Sprintf("/organizations/%s/environments/%s/agent", cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId)
		options.CloudApiKey = cfg.CloudContext.ApiKey
		options.CloudEnvironment = cfg.CloudContext.EnvironmentId
		options.CloudOrganization = cfg.CloudContext.OrganizationId
		options.ApiUri = cfg.CloudContext.ApiUri
	}

	c, err := client.GetClient(client.ClientType(clientType), options)
	if err != nil {
		return nil, "", fmt.Errorf("setting up client type %w", err)
	}

	return c, namespace, nil
}
