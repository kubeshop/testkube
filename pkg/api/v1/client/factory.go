package client

import (
	"fmt"

	"golang.org/x/oauth2"
)

type ClientType string

const (
	ClientDirect ClientType = "direct"
	ClientProxy  ClientType = "proxy"
)

// GetClient returns configured Testkube API client, can be one of direct and proxy - direct need additional proxy to be run (`make api-proxy`)
func GetClient(clientType ClientType, namespace, apiURI string, token *oauth2.Token, config *oauth2.Config) (client Client, err error) {
	switch clientType {
	case ClientDirect:
		var validToken *oauth2.Token
		if token != nil {
			source := oauth2.ReuseTokenSource(token, oauth2.StaticTokenSource(token))
			validToken, err = source.Token()
			if err != nil {
				return client, err
			}
		}

		client = NewDirectAPIClient(apiURI, validToken, config)
	case ClientProxy:
		clientset, err := GetClientSet("")
		if err != nil {
			return client, err
		}

		client = NewProxyAPIClient(clientset, NewAPIConfig(namespace))
	default:
		return client, fmt.Errorf("unsupported client type %s", clientType)
	}

	return client, err
}
