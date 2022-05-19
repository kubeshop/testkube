package client

import (
	"golang.org/x/oauth2"
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
func NewDirectAPIClient(apiURI string, token *oauth2.Token, config *oauth2.Config) APIClient {
	return APIClient{
		TestClient: NewTestClient(
			NewDirectTransport[testkube.Test](apiURI, token, config),
			NewDirectTransport[testkube.Execution](apiURI, token, config),
			NewDirectTransport[testkube.TestWithExecution](apiURI, token, config),
			NewDirectTransport[testkube.ExecutionsResult](apiURI, token, config),
			NewDirectTransport[testkube.Artifact](apiURI, token, config),
			NewDirectTransport[testkube.ServerInfo](apiURI, token, config),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewDirectTransport[testkube.TestSuite](apiURI, token, config),
			NewDirectTransport[testkube.TestSuiteExecution](apiURI, token, config),
			NewDirectTransport[testkube.TestSuiteWithExecution](apiURI, token, config),
			NewDirectTransport[testkube.TestSuiteExecutionsResult](apiURI, token, config),
		),
		ExecutorClient: NewExecutorClient(NewDirectTransport[testkube.ExecutorDetails](apiURI, token, config)),
		WebhookClient:  NewWebhookClient(NewDirectTransport[testkube.Webhook](apiURI, token, config)),
	}
}

// APIClient struct managing proxy API Client dependencies
type APIClient struct {
	TestClient
	TestSuiteClient
	ExecutorClient
	WebhookClient
}
