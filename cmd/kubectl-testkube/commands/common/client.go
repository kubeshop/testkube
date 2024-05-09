package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/cloudlogin"
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

	insecure, err := strconv.ParseBool(cmd.Flag("insecure").Value.String())
	if err != nil {
		return nil, "", fmt.Errorf("parsing flag value %w", err)
	}

	headers, err := cmd.Flags().GetStringToString("header")
	if err != nil {
		return nil, "", fmt.Errorf("parsing flag value %w", err)
	}

	options := client.Options{
		Namespace: namespace,
		ApiUri:    apiURI,
		Insecure:  insecure,
		Headers:   headers,
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

		token := cfg.CloudContext.ApiKey

		if cfg.CloudContext.ApiKey != "" && cfg.CloudContext.RefreshToken != "" && cfg.OAuth2Data.Enabled {
			var refreshToken string
			authURI := fmt.Sprintf("%s/idp", cfg.CloudContext.ApiUri)
			token, refreshToken, err = cloudlogin.CheckAndRefreshToken(context.Background(), authURI, cfg.CloudContext.ApiKey, cfg.CloudContext.RefreshToken)
			if err != nil {
				// Error: failed refreshing, go thru login flow
				token, refreshToken, err = LoginUser(authURI)
				if err != nil {
					return nil, "", fmt.Errorf("error logging in: %w", err)
				}
			}
			if err := UpdateTokens(cfg, token, refreshToken); err != nil {
				return nil, "", fmt.Errorf("error storing new token: %w", err)
			}
		}
		clientType = string(client.ClientCloud)
		options.CloudApiPathPrefix = fmt.Sprintf("/organizations/%s/environments/%s/agent", cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId)
		options.CloudApiKey = token
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
