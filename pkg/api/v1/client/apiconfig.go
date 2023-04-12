package client

func NewAPIConfig(namespace string) APIConfig {
	return APIConfig{
		Namespace:   namespace,
		ServiceName: "testkube-api-server",
		ServicePort: 8088,
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
