package tools

import "context"

type TestkubeClient interface {
	GetExecutionLogs(ctx context.Context, executionId string) (string, error)

	// Workflow methods
	ListWorkflows(ctx context.Context, params ListWorkflowsParams) (string, error)
	GetWorkflow(ctx context.Context, workflowName string) (string, error)
	GetWorkflowDefinition(ctx context.Context, workflowName string) (string, error)
	CreateWorkflow(ctx context.Context, workflowDefinition string) (string, error)
	RunWorkflow(ctx context.Context, params RunWorkflowParams) (string, error)

	// Execution methods
	GetExecutionInfo(ctx context.Context, workflowName, executionId string) (string, error)
	ListExecutions(ctx context.Context, params ListExecutionsParams) (string, error)
	ListArtifacts(ctx context.Context, workflowName, executionId string) (string, error)
	ReadArtifact(ctx context.Context, executionId, filename string) (string, error)
	LookupExecutionID(ctx context.Context, executionName string) (string, error)
}

type ListWorkflowsParams struct {
	ResourceGroup string
	Selector      string
	TextSearch    string
	PageSize      int
	Page          int
	Status        string
	GroupID       string
}

type ListExecutionsParams struct {
	WorkflowName string
	Selector     string
	TextSearch   string
	PageSize     int
	Page         int
	Status       string
	Since        string
}

type RunWorkflowParams struct {
	WorkflowName string
	Config       map[string]any
	Target       map[string]any
}

type GetClientFn func(ctx context.Context) (TestkubeClient, error)
