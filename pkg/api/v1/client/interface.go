package client

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Client is the Testkube API client abstraction
type Client interface {
	WebhookAPI
	WebhookTemplateAPI
	ServiceAPI
	ConfigAPI
	TestWorkflowAPI
	TestWorkflowExecutionAPI
	TestWorkflowTemplateAPI
	TestTriggerAPI
	SharedAPI
}

// WebhookAPI describes webhook api methods
type WebhookAPI interface {
	CreateWebhook(options CreateWebhookOptions) (webhook testkube.Webhook, err error)
	UpdateWebhook(options UpdateWebhookOptions) (webhook testkube.Webhook, err error)
	GetWebhook(name string) (webhook testkube.Webhook, err error)
	ListWebhooks(selector string) (webhooks testkube.Webhooks, err error)
	DeleteWebhook(name string) (err error)
	DeleteWebhooks(selector string) (err error)
}

// WebhookTemplateAPI describes webhook template api methods
type WebhookTemplateAPI interface {
	CreateWebhookTemplate(options CreateWebhookTemplateOptions) (webhookTemplate testkube.WebhookTemplate, err error)
	UpdateWebhookTemplate(options UpdateWebhookTemplateOptions) (webhookTemplate testkube.WebhookTemplate, err error)
	GetWebhookTemplate(name string) (webhookTemplate testkube.WebhookTemplate, err error)
	ListWebhookTemplates(selector string) (webhookTemplates testkube.WebhookTemplates, err error)
	DeleteWebhookTemplate(name string) (err error)
	DeleteWebhookTemplates(selector string) (err error)
}

// TestTriggerAPI describes test triggers api methods
type TestTriggerAPI interface {
	CreateTestTrigger(options CreateTestTriggerOptions) (testTrigger testkube.TestTrigger, err error)
	UpdateTestTrigger(options UpdateTestTriggerOptions) (testTrigger testkube.TestTrigger, err error)
	UpdateTestTriggerWithReplaceMode(options UpdateTestTriggerOptions) (testTrigger testkube.TestTrigger, err error)
	GetTestTrigger(name string) (testTrigger testkube.TestTrigger, err error)
	ListTestTriggers(selector string) (testTriggers []testkube.TestTrigger, err error)
	DeleteTestTrigger(name string) (err error)
	DeleteTestTriggers(selector string) (err error)
}

// ConfigAPI describes config api methods
type ConfigAPI interface {
	UpdateConfig(config testkube.Config) (outputConfig testkube.Config, err error)
	GetConfig() (config testkube.Config, err error)
}

// ServiceAPI describes service api methods
type ServiceAPI interface {
	GetServerInfo() (info testkube.ServerInfo, err error)
	GetDebugInfo() (info testkube.DebugInfo, err error)
}

type SharedAPI interface {
	ListLabels() (labels map[string][]string, err error)
}

// TestWorkflowAPI describes test workflow api methods
type TestWorkflowAPI interface {
	GetTestWorkflow(id string) (testkube.TestWorkflow, error)
	GetTestWorkflowWithExecution(id string) (testkube.TestWorkflowWithExecution, error)
	ListTestWorkflows(selector string) (testkube.TestWorkflows, error)
	ListTestWorkflowWithExecutions(selector string) (testkube.TestWorkflowWithExecutions, error)
	DeleteTestWorkflows(selector string) error
	CreateTestWorkflow(workflow testkube.TestWorkflow) (testkube.TestWorkflow, error)
	UpdateTestWorkflow(workflow testkube.TestWorkflow) (testkube.TestWorkflow, error)
	UpdateTestWorkflowStatus(workflow testkube.TestWorkflow) error
	DeleteTestWorkflow(name string) error
	ExecuteTestWorkflow(name string, request testkube.TestWorkflowExecutionRequest) (testkube.TestWorkflowExecution, error)
	ExecuteTestWorkflows(selector string, request testkube.TestWorkflowExecutionRequest) ([]testkube.TestWorkflowExecution, error)
	GetTestWorkflowExecutionNotifications(id string) (chan testkube.TestWorkflowExecutionNotification, error)
	GetTestWorkflowExecutionLogs(id string) ([]byte, error)
	GetTestWorkflowExecutionServiceNotifications(id, serviceName string, serviceIndex int) (chan testkube.TestWorkflowExecutionNotification, error)
	GetTestWorkflowExecutionParallelStepNotifications(id, ref string, workerIndex int) (chan testkube.TestWorkflowExecutionNotification, error)
}

// TestWorkflowExecutionAPI describes test workflow api methods
type TestWorkflowExecutionAPI interface {
	GetTestWorkflowExecution(executionID string) (execution testkube.TestWorkflowExecution, err error)
	ListTestWorkflowExecutions(id string, limit int, options FilterTestWorkflowExecutionOptions) (executions testkube.TestWorkflowExecutionsResult, err error)
	AbortTestWorkflowExecution(workflow string, id string, force bool) error
	AbortTestWorkflowExecutions(workflow string) error
	PauseTestWorkflowExecution(workflow string, id string) error
	ResumeTestWorkflowExecution(workflow string, id string) error
	GetTestWorkflowExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
	DownloadTestWorkflowArtifact(executionID, fileName, destination string) (artifact string, err error)
	DownloadTestWorkflowArtifactArchive(executionID, destination string, masks []string) (archive string, err error)
	ReRunTestWorkflowExecution(workflow string, id string, runningContext *testkube.TestWorkflowRunningContext) (testkube.TestWorkflowExecution, error)
	ValidateTestWorkflow(body []byte) error
}

// TestWorkflowTemplateAPI describes test workflow api methods
type TestWorkflowTemplateAPI interface {
	GetTestWorkflowTemplate(id string) (testkube.TestWorkflowTemplate, error)
	ListTestWorkflowTemplates(selector string) (testkube.TestWorkflowTemplates, error)
	DeleteTestWorkflowTemplates(selector string) error
	CreateTestWorkflowTemplate(workflow testkube.TestWorkflowTemplate) (testkube.TestWorkflowTemplate, error)
	UpdateTestWorkflowTemplate(workflow testkube.TestWorkflowTemplate) (testkube.TestWorkflowTemplate, error)
	DeleteTestWorkflowTemplate(name string) error
	ValidateTestWorkflowTemplate(body []byte) error
}

// CreateWebhookOptions - is mapping for now to OpenAPI schema for creating/changing webhook
type CreateWebhookOptions testkube.WebhookCreateRequest

// UpdateWebhookOptions - is mapping for now to OpenAPI schema for changing webhook request
type UpdateWebhookOptions testkube.WebhookUpdateRequest

// CreateWebhookTemplateOptions - is mapping for now to OpenAPI schema for creating/changing webhook template
type CreateWebhookTemplateOptions testkube.WebhookTemplateCreateRequest

// UpdateWebhookTemplateOptions - is mapping for now to OpenAPI schema for changing webhook template request
type UpdateWebhookTemplateOptions testkube.WebhookTemplateUpdateRequest

// CreateTestTriggerOptions - is mapping for now to OpenAPI schema for creating trigger
type CreateTestTriggerOptions testkube.TestTriggerUpsertRequest

// UpdateTestTriggerOptions - is mapping for now to OpenAPI schema for changing trigger request
type UpdateTestTriggerOptions testkube.TestTriggerUpsertRequest

// FilterTestWorkflowExecutionOptions contains filter test workflow execution options
type FilterTestWorkflowExecutionOptions struct {
	Selector    string
	TagSelector string
	ActorName   string
	ActorType   testkube.TestWorkflowRunningContextActorType
	Status      string
}

// Gettable is an interface of gettable objects
type Gettable interface {
	testkube.Webhook | testkube.Artifact | testkube.ServerInfo | testkube.Config | testkube.DebugInfo |
		testkube.TestWorkflow | testkube.TestWorkflowWithExecution | testkube.TestWorkflowTemplate | testkube.TestWorkflowExecution |
		testkube.TestTrigger | testkube.WebhookTemplate | map[string][]string
}

// Executable is an interface of executable objects
type Executable interface {
	testkube.TestWorkflowExecution | testkube.TestWorkflowExecutionsResult
}

// All is an interface of all objects
type All interface {
	Gettable | Executable
}

// Transport provides methods to execute api calls
type Transport[A All] interface {
	Execute(method, uri string, body []byte, params map[string]string) (result A, err error)
	ExecuteMultiple(method, uri string, body []byte, params map[string]string) (result []A, err error)
	Delete(uri, selector string, isContentExpected bool) error
	ExecuteMethod(method, uri string, params map[string]string, isContentExpected bool) error
	GetURI(pathTemplate string, params ...interface{}) string
	GetTestWorkflowExecutionNotifications(uri string, notifications chan testkube.TestWorkflowExecutionNotification) error
	GetFile(uri, fileName, destination string, params map[string][]string) (name string, err error)
	GetRawBody(method, uri string, body []byte, params map[string]string) (result []byte, err error)
	Validate(method, uri string, body []byte, params map[string]string) error
}
