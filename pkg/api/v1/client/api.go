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
			NewProxyClient[testkube.Test](client, config),
			NewProxyClient[testkube.Execution](client, config),
			NewProxyClient[testkube.TestWithExecution](client, config),
			NewProxyClient[testkube.ExecutionsResult](client, config),
			NewProxyClient[testkube.Artifact](client, config),
			NewProxyClient[testkube.ServerInfo](client, config),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewProxyClient[testkube.TestSuite](client, config),
			NewProxyClient[testkube.TestSuiteExecution](client, config),
			NewProxyClient[testkube.TestSuiteWithExecution](client, config),
			NewProxyClient[testkube.TestSuiteExecutionsResult](client, config),
		),
		ExecutorClient: NewExecutorClient(NewProxyClient[testkube.ExecutorDetails](client, config)),
		WebhookClient:  NewWebhookClient(NewProxyClient[testkube.Webhook](client, config)),
	}
}

// NewDirectAPIClient returns direct api client
func NewDirectAPIClient(apiURI string, token *oauth2.Token, config *oauth2.Config) APIClient {
	return APIClient{
		TestClient: NewTestClient(
			NewDirectClient[testkube.Test](apiURI, token, config),
			NewDirectClient[testkube.Execution](apiURI, token, config),
			NewDirectClient[testkube.TestWithExecution](apiURI, token, config),
			NewDirectClient[testkube.ExecutionsResult](apiURI, token, config),
			NewDirectClient[testkube.Artifact](apiURI, token, config),
			NewDirectClient[testkube.ServerInfo](apiURI, token, config),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewDirectClient[testkube.TestSuite](apiURI, token, config),
			NewDirectClient[testkube.TestSuiteExecution](apiURI, token, config),
			NewDirectClient[testkube.TestSuiteWithExecution](apiURI, token, config),
			NewDirectClient[testkube.TestSuiteExecutionsResult](apiURI, token, config),
		),
		ExecutorClient: NewExecutorClient(NewDirectClient[testkube.ExecutorDetails](apiURI, token, config)),
		WebhookClient:  NewWebhookClient(NewDirectClient[testkube.Webhook](apiURI, token, config)),
	}
}

// APIClient struct managing proxy API Client dependencies
type APIClient struct {
	TestClient
	TestSuiteClient
	ExecutorClient
	WebhookClient
}
