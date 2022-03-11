package client

import (
	"os"

	"k8s.io/client-go/kubernetes"
)

type ClientType string

const (
	ClientDirect ClientType = "direct"
	ClientProxy  ClientType = "proxy"
)

// GetClient returns configured Testkube API client, can be one of direct and proxy - direct need additional proxy to be run (`make api-proxy`)
func GetClient(clientType ClientType, namespace string) (client Client, err error) {
	var overrideHost string
	var clientset kubernetes.Interface

	if clientType == ClientDirect {
		overrideHost = "http://127.0.0.1:8080"
		if host, ok := os.LookupEnv("TESTKUBE_KUBEPROXY_HOST"); ok {
			overrideHost = host
		}
	}

	clientset, err = GetClientSet(overrideHost)
	if err != nil {
		return client, err
	}

	client = NewAPIClient(clientset, NewAPIConfig(namespace))

	return client, err
}
