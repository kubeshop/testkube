package client

import (
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// check in compile time if interface is implemented
var _ Client = (*APIClient)(nil)

// NewProxyAPIClient returns proxy api client
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

// NewDirectAPIClient returns direct api client
func NewDirectAPIClient(apiURI string) APIClient {
	return APIClient{
		TestClient: NewTestClient(
			NewDirectTransport[testkube.Test](apiURI),
			NewDirectTransport[testkube.Execution](apiURI),
			NewDirectTransport[testkube.TestWithExecution](apiURI),
			NewDirectTransport[testkube.ExecutionsResult](apiURI),
			NewDirectTransport[testkube.Artifact](apiURI),
			NewDirectTransport[testkube.ServerInfo](apiURI),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewDirectTransport[testkube.TestSuite](apiURI),
			NewDirectTransport[testkube.TestSuiteExecution](apiURI),
			NewDirectTransport[testkube.TestSuiteWithExecution](apiURI),
			NewDirectTransport[testkube.TestSuiteExecutionsResult](apiURI),
		),
		ExecutorClient: NewExecutorClient(NewDirectTransport[testkube.ExecutorDetails](apiURI)),
		WebhookClient:  NewWebhookClient(NewDirectTransport[testkube.Webhook](apiURI)),
	}
}

// APIClient struct managing proxy API Client dependencies
type APIClient struct {
	TestClient
	TestSuiteClient
	ExecutorClient
	WebhookClient
}
