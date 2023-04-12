package client

func NewAPIConfig(namespace, serviceName string, servicePort int) APIConfig {
	return APIConfig{
		Namespace:   namespace,
		ServiceName: serviceName,
		ServicePort: servicePort,
	}
}

type APIConfig struct {
	// Namespace where testkube is installed
	Namespace string
	// API Server service name
	ServiceName string
	// API Server service port
	ServicePort int
}
