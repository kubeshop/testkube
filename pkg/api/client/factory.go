package client

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
		client = NewProxyScriptsAPI(clientset, NewProxyConfig(namespace))
	}

	return client, err
}
