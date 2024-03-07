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
			NewProxyClient[testkube.TestWithExecutionSummary](client, config),
			NewProxyClient[testkube.ExecutionsResult](client, config),
			NewProxyClient[testkube.Artifact](client, config),
			NewProxyClient[testkube.ServerInfo](client, config),
			NewProxyClient[testkube.DebugInfo](client, config),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewProxyClient[testkube.TestSuite](client, config),
			NewProxyClient[testkube.TestSuiteExecution](client, config),
			NewProxyClient[testkube.TestSuiteWithExecution](client, config),
			NewProxyClient[testkube.TestSuiteWithExecutionSummary](client, config),
			NewProxyClient[testkube.TestSuiteExecutionsResult](client, config),
			NewProxyClient[testkube.Artifact](client, config),
		),
		ExecutorClient:   NewExecutorClient(NewProxyClient[testkube.ExecutorDetails](client, config)),
		WebhookClient:    NewWebhookClient(NewProxyClient[testkube.Webhook](client, config)),
		ConfigClient:     NewConfigClient(NewProxyClient[testkube.Config](client, config)),
		TestSourceClient: NewTestSourceClient(NewProxyClient[testkube.TestSource](client, config)),
		CopyFileClient:   NewCopyFileProxyClient(client, config),
		TemplateClient:   NewTemplateClient(NewProxyClient[testkube.Template](client, config)),
		TestWorkflowClient: NewTestWorkflowClient(
			NewProxyClient[testkube.TestWorkflow](client, config),
			NewProxyClient[testkube.TestWorkflowWithExecution](client, config),
			NewProxyClient[testkube.TestWorkflowExecution](client, config),
			NewProxyClient[testkube.TestWorkflowExecutionsResult](client, config),
			NewProxyClient[testkube.Artifact](client, config),
		),
		TestWorkflowTemplateClient: NewTestWorkflowTemplateClient(NewProxyClient[testkube.TestWorkflowTemplate](client, config)),
	}
}

// NewDirectAPIClient returns direct api client
func NewDirectAPIClient(httpClient *http.Client, sseClient *http.Client, apiURI, apiPathPrefix string) APIClient {
	return APIClient{
		TestClient: NewTestClient(
			NewDirectClient[testkube.Test](httpClient, apiURI, apiPathPrefix).WithSSEClient(sseClient),
			NewDirectClient[testkube.Execution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWithExecution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWithExecutionSummary](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.ExecutionsResult](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.ServerInfo](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.DebugInfo](httpClient, apiURI, apiPathPrefix),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewDirectClient[testkube.TestSuite](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestSuiteExecution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestSuiteWithExecution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestSuiteWithExecutionSummary](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestSuiteExecutionsResult](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix),
		),
		ExecutorClient:   NewExecutorClient(NewDirectClient[testkube.ExecutorDetails](httpClient, apiURI, apiPathPrefix)),
		WebhookClient:    NewWebhookClient(NewDirectClient[testkube.Webhook](httpClient, apiURI, apiPathPrefix)),
		ConfigClient:     NewConfigClient(NewDirectClient[testkube.Config](httpClient, apiURI, apiPathPrefix)),
		TestSourceClient: NewTestSourceClient(NewDirectClient[testkube.TestSource](httpClient, apiURI, apiPathPrefix)),
		CopyFileClient:   NewCopyFileDirectClient(httpClient, apiURI, apiPathPrefix),
		TemplateClient:   NewTemplateClient(NewDirectClient[testkube.Template](httpClient, apiURI, apiPathPrefix)),
		TestWorkflowClient: NewTestWorkflowClient(
			NewDirectClient[testkube.TestWorkflow](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWorkflowWithExecution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWorkflowExecution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWorkflowExecutionsResult](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix),
		),
		TestWorkflowTemplateClient: NewTestWorkflowTemplateClient(NewDirectClient[testkube.TestWorkflowTemplate](httpClient, apiURI, apiPathPrefix)),
	}
}

// NewCloudAPIClient returns cloud api client
func NewCloudAPIClient(httpClient *http.Client, sseClient *http.Client, apiURI, apiPathPrefix string) APIClient {
	return APIClient{
		TestClient: NewTestClient(
			NewCloudClient[testkube.Test](httpClient, apiURI, apiPathPrefix).WithSSEClient(sseClient),
			NewCloudClient[testkube.Execution](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestWithExecution](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestWithExecutionSummary](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.ExecutionsResult](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.ServerInfo](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.DebugInfo](httpClient, apiURI, apiPathPrefix),
		),
		TestSuiteClient: NewTestSuiteClient(
			NewCloudClient[testkube.TestSuite](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestSuiteExecution](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestSuiteWithExecution](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestSuiteWithExecutionSummary](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestSuiteExecutionsResult](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix),
		),
		ExecutorClient:   NewExecutorClient(NewCloudClient[testkube.ExecutorDetails](httpClient, apiURI, apiPathPrefix)),
		WebhookClient:    NewWebhookClient(NewCloudClient[testkube.Webhook](httpClient, apiURI, apiPathPrefix)),
		ConfigClient:     NewConfigClient(NewCloudClient[testkube.Config](httpClient, apiURI, apiPathPrefix)),
		TestSourceClient: NewTestSourceClient(NewCloudClient[testkube.TestSource](httpClient, apiURI, apiPathPrefix)),
		CopyFileClient:   NewCopyFileDirectClient(httpClient, apiURI, apiPathPrefix),
		TemplateClient:   NewTemplateClient(NewCloudClient[testkube.Template](httpClient, apiURI, apiPathPrefix)),
		TestWorkflowClient: NewTestWorkflowClient(
			NewCloudClient[testkube.TestWorkflow](httpClient, apiURI, apiPathPrefix).WithSSEClient(sseClient),
			NewCloudClient[testkube.TestWorkflowWithExecution](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestWorkflowExecution](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.TestWorkflowExecutionsResult](httpClient, apiURI, apiPathPrefix),
			NewCloudClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix),
		),
		TestWorkflowTemplateClient: NewTestWorkflowTemplateClient(NewCloudClient[testkube.TestWorkflowTemplate](httpClient, apiURI, apiPathPrefix)),
	}
}

// APIClient struct managing proxy API Client dependencies
type APIClient struct {
	TestClient
	TestSuiteClient
	ExecutorClient
	WebhookClient
	ConfigClient
	TestSourceClient
	CopyFileClient
	TemplateClient
	TestWorkflowClient
	TestWorkflowTemplateClient
}
