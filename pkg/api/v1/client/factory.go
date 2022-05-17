package client

type ClientType string

const (
	ClientDirect ClientType = "direct"
	ClientProxy  ClientType = "proxy"
)

// GetClient returns configured Testkube API client, can be one of direct and proxy - direct need additional proxy to be run (`make api-proxy`)
func GetClient(clientType ClientType, namespace, host string) (client Client, err error) {
	if clientType == ClientDirect {
		client = NewDirectAPIClient(host)
	}

	if clientType == ClientProxy {
		clientset, err := GetClientSet(host)
		if err != nil {
			return client, err
		}

		client = NewProxyAPIClient(clientset, NewAPIConfig(namespace))
	}

	return client, err
}
