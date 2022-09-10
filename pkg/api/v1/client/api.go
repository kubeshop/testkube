package client

import (
	"net/http"

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
			NewProxyClient[testkube.DebugInfo](client, config),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewProxyClient[testkube.TestSuite](client, config),
			NewProxyClient[testkube.TestSuiteExecution](client, config),
			NewProxyClient[testkube.TestSuiteWithExecution](client, config),
			NewProxyClient[testkube.TestSuiteExecutionsResult](client, config),
		),
		ExecutorClient: NewExecutorClient(NewProxyClient[testkube.ExecutorDetails](client, config)),
		WebhookClient:  NewWebhookClient(NewProxyClient[testkube.Webhook](client, config)),
		ConfigClient:   NewConfigClient(NewProxyClient[testkube.Config](client, config)),
	}
}

// NewDirectAPIClient returns direct api client
func NewDirectAPIClient(httpClient *http.Client, apiURI string) APIClient {
	return APIClient{
		TestClient: NewTestClient(
			NewDirectClient[testkube.Test](httpClient, apiURI),
			NewDirectClient[testkube.Execution](httpClient, apiURI),
			NewDirectClient[testkube.TestWithExecution](httpClient, apiURI),
			NewDirectClient[testkube.ExecutionsResult](httpClient, apiURI),
			NewDirectClient[testkube.Artifact](httpClient, apiURI),
			NewDirectClient[testkube.ServerInfo](httpClient, apiURI),
			NewDirectClient[testkube.DebugInfo](httpClient, apiURI),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewDirectClient[testkube.TestSuite](httpClient, apiURI),
			NewDirectClient[testkube.TestSuiteExecution](httpClient, apiURI),
			NewDirectClient[testkube.TestSuiteWithExecution](httpClient, apiURI),
			NewDirectClient[testkube.TestSuiteExecutionsResult](httpClient, apiURI),
		),
		ExecutorClient: NewExecutorClient(NewDirectClient[testkube.ExecutorDetails](httpClient, apiURI)),
		WebhookClient:  NewWebhookClient(NewDirectClient[testkube.Webhook](httpClient, apiURI)),
		ConfigClient:   NewConfigClient(NewDirectClient[testkube.Config](httpClient, apiURI)),
	}
}

// APIClient struct managing proxy API Client dependencies
type APIClient struct {
	TestClient
	TestSuiteClient
	ExecutorClient
	WebhookClient
	ConfigClient
}
