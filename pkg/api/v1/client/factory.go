package client

import (
	"fmt"

	"golang.org/x/oauth2"

	phttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/oauth"
)

type ClientType string

const (
	ClientDirect  ClientType = "direct"
	ClientCloud   ClientType = "cloud"
	ClientProxy   ClientType = "proxy"
	ClientCluster ClientType = "cluster"
)

// Options contains client options
type Options struct {
	Namespace     string
	ApiUri        string
	ApiPath       string
	Token         *oauth2.Token
	Provider      oauth.ProviderType
	ClientID      string
	ClientSecret  string
	Scopes        []string
	APIServerName string
	APIServerPort int
	Insecure      bool
	Headers       map[string]string

	// Testkube Cloud
	CloudApiPathPrefix string
	CloudApiKey        string
	CloudOrganization  string
	CloudEnvironment   string
}

// GetClient returns configured Testkube API client, can be one of direct and proxy - direct need additional proxy to be run (`make api-proxy`)
func GetClient(clientType ClientType, options Options) (client Client, err error) {
	httpClient := phttp.NewClient(options.Insecure)
	sseClient := phttp.NewSSEClient(options.Insecure)

	switch clientType {

	case ClientCloud:
		ConfigureClient(httpClient, nil, options.CloudApiKey, options.Headers)
		ConfigureClient(sseClient, nil, options.CloudApiKey, options.Headers)
		client = NewCloudAPIClient(httpClient, sseClient, options.ApiUri, options.CloudApiPathPrefix)

	case ClientDirect:
		var token *oauth2.Token
		if options.Token != nil {
			provider := oauth.NewProvider(options.ClientID, options.ClientSecret, options.Scopes)
			if token, err = provider.ValidateToken(options.Provider, options.Token); err != nil {
				return client, err
			}
		}

		ConfigureClient(httpClient, token, "", options.Headers)
		ConfigureClient(sseClient, token, "", options.Headers)
		client = NewDirectAPIClient(httpClient, sseClient, options.ApiUri, "")

	case ClientProxy, ClientCluster:
		clientset, err := GetClientSet("", clientType)
		if err != nil {
			return client, err
		}

		client = NewProxyAPIClient(clientset, NewAPIConfig(options.Namespace, options.APIServerName, options.APIServerPort))
	default:
		return client, fmt.Errorf("unsupported client type %s", clientType)
	}

	return client, err
}
