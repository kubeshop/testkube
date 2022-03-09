package client

import (
	"k8s.io/client-go/kubernetes"
)

type ClientType string

const (
	ClientDirect ClientType = "direct"
	ClientProxy  ClientType = "proxy"
)

func GetClient(clientType ClientType, namespace string) (client Client, err error) {
	var overrideHost string
	var clientset kubernetes.Interface

	if clientType == ClientDirect {
		overrideHost = "http://127.0.0.1:8080"
	}

	clientset, err = GetClientSet(overrideHost)
	if err != nil {
		return client, err
	}

	client = NewProxyAPIClient(clientset, NewProxyConfig(namespace))

	return client, err
}
