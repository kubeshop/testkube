package common

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/cloudlogin"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
)

const UserAgentCLI = "Testkube-CLI"

// GetClient returns api client
func GetClient(cmd *cobra.Command) (client.Client, string, error) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	apiURI := cmd.Flag("api-uri").Value.String()

	insecure, err := strconv.ParseBool(cmd.Flag("insecure").Value.String())
	if err != nil {
		return nil, "", fmt.Errorf("parsing flag value %w", err)
	}

	headers, err := cmd.Flags().GetStringToString("header")
	if err != nil {
		return nil, "", fmt.Errorf("parsing flag value %w", err)
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["User-Agent"] = userAgent()

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

	if cfg.ContextType == config.ContextTypeCloud {
		token := cfg.CloudContext.ApiKey

		if cfg.CloudContext.ApiKey != "" && cfg.CloudContext.RefreshToken != "" {
			newTokenType := cfg.CloudContext.TokenType
			var refreshToken string
			token, refreshToken, err = refreshUserToken(context.Background(), cfg)
			if err != nil {
				if cfg.CloudContext.TokenType == config.TokenTypeEmailLink {
					// Don't auto-restart the email-link flow from inside an
					// unrelated command — a 5-minute "check your inbox" wait
					// mid-command is worse UX than a hard error. Surface the
					// exact command to re-run, filling in the user's email
					// from the stored ID token when we can recover it.
					hint := "testkube pro login --email-link <email>"
					if stored := cloudlogin.EmailFromIDToken(cfg.CloudContext.ApiKey); stored != "" {
						hint = fmt.Sprintf("testkube pro login --email-link %s", stored)
					}
					return nil, "", fmt.Errorf("email-link token refresh failed; re-run `%s`: %w", hint, err)
				}
				authURI := cfg.CloudContext.AuthUri
				if authURI == "" {
					authURI = fmt.Sprintf("%s/idp", cfg.CloudContext.ApiUri)
				}
				port := config.CallbackPort
				if cfg.CloudContext.CallbackPort != 0 {
					port = cfg.CloudContext.CallbackPort
				}
				newTokenType, token, refreshToken, err = LoginUser(authURI, cfg.CloudContext.ApiUri, cfg.CloudContext.CustomAuth, port)
				if err != nil {
					return nil, "", fmt.Errorf("error logging in: %w", err)
				}
			}
			if err := UpdateTokens(cfg, newTokenType, token, refreshToken); err != nil {
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

func userAgent() string {
	return fmt.Sprintf("%s/%s (%s; %s) Go/%s", UserAgentCLI, Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
