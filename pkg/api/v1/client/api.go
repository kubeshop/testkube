package client

import (
	"net/http"

	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewProxyAPIClient returns proxy api client
func NewProxyAPIClient(client kubernetes.Interface, config APIConfig) APIClient {
	return APIClient{
		WebhookClient:         NewWebhookClient(NewProxyClient[testkube.Webhook](client, config)),
		WebhookTemplateClient: NewWebhookTemplateClient(NewProxyClient[testkube.WebhookTemplate](client, config)),
		ConfigClient:          NewConfigClient(NewProxyClient[testkube.Config](client, config)),
		TestWorkflowClient: NewTestWorkflowClient(
			NewProxyClient[testkube.TestWorkflow](client, config),
			NewProxyClient[testkube.TestWorkflowWithExecution](client, config),
			NewProxyClient[testkube.TestWorkflowExecution](client, config),
			NewProxyClient[testkube.TestWorkflowExecutionsResult](client, config),
			NewProxyClient[testkube.Artifact](client, config),
		),
		TestWorkflowTemplateClient: NewTestWorkflowTemplateClient(NewProxyClient[testkube.TestWorkflowTemplate](client, config)),
		TestTriggerClient:          NewTestTriggerClient(NewProxyClient[testkube.TestTrigger](client, config)),
		WorkflowTriggerClient:      NewWorkflowTriggerClient(NewProxyClient[testkube.WorkflowTrigger](client, config)),
		SharedClient: NewSharedClient(
			NewProxyClient[map[string][]string](client, config),
			NewProxyClient[testkube.ServerInfo](client, config),
			NewProxyClient[testkube.DebugInfo](client, config),
		),
	}
}

// NewDirectAPIClient returns direct api client
func NewDirectAPIClient(httpClient *http.Client, sseClient *http.Client, apiURI, apiPathPrefix string) APIClient {
	return APIClient{
		WebhookClient:         NewWebhookClient(NewDirectClient[testkube.Webhook](httpClient, apiURI, apiPathPrefix)),
		WebhookTemplateClient: NewWebhookTemplateClient(NewDirectClient[testkube.WebhookTemplate](httpClient, apiURI, apiPathPrefix)),
		ConfigClient:          NewConfigClient(NewDirectClient[testkube.Config](httpClient, apiURI, apiPathPrefix)),
		TestWorkflowClient: NewTestWorkflowClient(
			NewDirectClient[testkube.TestWorkflow](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWorkflowWithExecution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWorkflowExecution](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.TestWorkflowExecutionsResult](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix),
		),
		TestWorkflowTemplateClient: NewTestWorkflowTemplateClient(NewDirectClient[testkube.TestWorkflowTemplate](httpClient, apiURI, apiPathPrefix)),
		TestTriggerClient:          NewTestTriggerClient(NewDirectClient[testkube.TestTrigger](httpClient, apiURI, apiPathPrefix)),
		WorkflowTriggerClient:      NewWorkflowTriggerClient(NewDirectClient[testkube.WorkflowTrigger](httpClient, apiURI, apiPathPrefix)),
		SharedClient: NewSharedClient(
			NewDirectClient[map[string][]string](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.ServerInfo](httpClient, apiURI, apiPathPrefix),
			NewDirectClient[testkube.DebugInfo](httpClient, apiURI, apiPathPrefix),
		),
	}
}

// NewCloudAPIClient returns cloud api client
func NewCloudAPIClient(httpClient *http.Client, sseClient *http.Client, apiURI, apiPathPrefix string, insecure ...bool) APIClient {
	return APIClient{
		WebhookClient:         NewWebhookClient(NewCloudClient[testkube.Webhook](httpClient, apiURI, apiPathPrefix, insecure...)),
		WebhookTemplateClient: NewWebhookTemplateClient(NewCloudClient[testkube.WebhookTemplate](httpClient, apiURI, apiPathPrefix, insecure...)),
		ConfigClient:          NewConfigClient(NewCloudClient[testkube.Config](httpClient, apiURI, apiPathPrefix, insecure...)),
		TestWorkflowClient: NewTestWorkflowClient(
			NewCloudClient[testkube.TestWorkflow](httpClient, apiURI, apiPathPrefix, insecure...).WithSSEClient(sseClient),
			NewCloudClient[testkube.TestWorkflowWithExecution](httpClient, apiURI, apiPathPrefix, insecure...),
			NewCloudClient[testkube.TestWorkflowExecution](httpClient, apiURI, apiPathPrefix, insecure...),
			NewCloudClient[testkube.TestWorkflowExecutionsResult](httpClient, apiURI, apiPathPrefix, insecure...),
			NewCloudClient[testkube.Artifact](httpClient, apiURI, apiPathPrefix, insecure...),
		),
		TestWorkflowTemplateClient: NewTestWorkflowTemplateClient(NewCloudClient[testkube.TestWorkflowTemplate](httpClient, apiURI, apiPathPrefix, insecure...)),
		TestTriggerClient:          NewTestTriggerClient(NewCloudClient[testkube.TestTrigger](httpClient, apiURI, apiPathPrefix, insecure...)),
		WorkflowTriggerClient:      NewWorkflowTriggerClient(NewCloudClient[testkube.WorkflowTrigger](httpClient, apiURI, apiPathPrefix, insecure...)),
		SharedClient: NewSharedClient(
			NewCloudClient[map[string][]string](httpClient, apiURI, apiPathPrefix, insecure...),
			NewCloudClient[testkube.ServerInfo](httpClient, apiURI, apiPathPrefix, insecure...),
			NewCloudClient[testkube.DebugInfo](httpClient, apiURI, apiPathPrefix, insecure...),
		),
	}
}

// APIClient struct managing proxy API Client dependencies
type APIClient struct {
	WebhookClient
	WebhookTemplateClient
	ConfigClient
	TestWorkflowClient
	TestWorkflowTemplateClient
	TestTriggerClient
	WorkflowTriggerClient
	SharedClient
}

// check in compile time if interface is implemented
var _ Client = (*APIClient)(nil)
