package client

import "fmt"

type ClientType string

const (
	ClientDirect ClientType = "direct"
	ClientProxy  ClientType = "proxy"
)

func GetClient(clientType ClientType, namespace string) (client Client, err error) {
	switch clientType {

	case ClientDirect:
		client = NewDefaultDirectScriptsAPI()
	case ClientProxy:
		clientset, err := GetClientSet()
		if err != nil {
			return client, err
		}
		client = NewProxyAPIClient(clientset, NewProxyConfig(namespace))
	default:
		err = fmt.Errorf("Client %s is not handled by testkube, use one of: %v", clientType, []ClientType{ClientDirect, ClientProxy})
	}

	return client, err
}
