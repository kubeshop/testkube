package client

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/oauth"
	"golang.org/x/oauth2"
)

type ClientType string

const (
	ClientDirect ClientType = "direct"
	ClientProxy  ClientType = "proxy"
)

// Options contains client options
type Options struct {
	Namespace    string
	APIURI       string
	Token        *oauth2.Token
	Provider     oauth.ProviderType
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// GetClient returns configured Testkube API client, can be one of direct and proxy - direct need additional proxy to be run (`make api-proxy`)
func GetClient(clientType ClientType, options Options) (client Client, err error) {
	switch clientType {
	case ClientDirect:
		var token *oauth2.Token
		if options.Token != nil {
			provider := oauth.NewProvider(options.ClientID, options.ClientSecret, options.Scopes)
			if token, err = provider.ValidateToken(options.Provider, options.Token); err != nil {
				return client, err
			}
		}

		httpClient, err := GetHTTTPClient(token)
		if err != nil {
			return client, err
		}

		httpSSEClient, err := GetHTTTPSSEClient(token)
		if err != nil {
			return client, err
		}

		client = NewDirectAPIClient(httpClient, httpSSEClient, options.APIURI)
	case ClientProxy:
		clientset, err := GetClientSet("")
		if err != nil {
			return client, err
		}

		client = NewProxyAPIClient(clientset, NewAPIConfig(options.Namespace))
	default:
		return client, fmt.Errorf("unsupported client type %s", clientType)
	}

	return client, err
}
