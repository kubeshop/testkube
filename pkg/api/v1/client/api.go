package client

import (
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// check in compile time if interface is implemented
var _ Client = (*APIClient)(nil)

// NewProxyAPIClient returns
func NewProxyAPIClient(client kubernetes.Interface, config APIConfig) APIClient {
	return APIClient{
		TestClient: NewTestClient(
			NewProxyTransport[testkube.Test](client, config),
			NewProxyTransport[testkube.Execution](client, config),
			NewProxyTransport[testkube.TestWithExecution](client, config),
			NewProxyTransport[testkube.ExecutionsResult](client, config),
			NewProxyTransport[testkube.Artifact](client, config),
			NewProxyTransport[testkube.ServerInfo](client, config),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewProxyTransport[testkube.TestSuite](client, config),
			NewProxyTransport[testkube.TestSuiteExecution](client, config),
			NewProxyTransport[testkube.TestSuiteWithExecution](client, config),
			NewProxyTransport[testkube.TestSuiteExecutionsResult](client, config),
		),
		ExecutorClient: NewExecutorClient(NewProxyTransport[testkube.ExecutorDetails](client, config)),
		WebhookClient:  NewWebhookClient(NewProxyTransport[testkube.Webhook](client, config)),
	}
}

// APIClient struct managing proxy API Client dependencies
type APIClient struct {
	TestClient
	TestSuiteClient
	ExecutorClient
	WebhookClient
}
