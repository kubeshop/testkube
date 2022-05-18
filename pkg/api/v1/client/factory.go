package client

import "fmt"

type ClientType string

const (
	ClientDirect ClientType = "direct"
	ClientProxy  ClientType = "proxy"
)

// GetClient returns configured Testkube API client, can be one of direct and proxy - direct need additional proxy to be run (`make api-proxy`)
func GetClient(clientType ClientType, namespace, apiURI string) (client Client, err error) {
	switch clientType {
	case ClientDirect:
		client = NewDirectAPIClient(apiURI)
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
