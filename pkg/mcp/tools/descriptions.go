package tools

const (
	WorkflowNameDescription = `The name of the workflow. Workflow names are lowercase alphanumeric with dashes 
(e.g., 'my-workflow', 'api-tests'). This uniquely identifies a TestWorkflow within the organization.`

	ExecutionIdDescription = `The unique execution ID in MongoDB format (e.g., '67d2cdbc351aecb2720afdf2'). 
This is the internal identifier used by most tools that operate on specific executions. 
If you only have an execution name, use the lookup_execution_id tool first to get the ID.`

	ExecutionNameDescription = `The name of the execution (e.g., 'my-workflow-123'). Execution names follow 
the pattern of workflow name plus a numeric suffix. Use this when you have an execution name 
but need the execution ID for other operations.`

	PageDescription = "Page number for pagination (default: 1)"

	PageSizeDescription = "Number of items to return per page (default: 10, max: 100)"

	TextSearchDescription = `Text search filter for names or descriptions. Can use space-separated words 
to find items containing all terms`

	SelectorDescription = "Kubernetes-style label selector for filtering"

	StatusDescription = `Filter by execution status. Available statuses: 'queued', 'running', 'passed', 
'failed', 'skipped', 'aborted', 'timeout', 'paused'`

	ResourceGroupDescription = "Filter by resource group"

	GroupIdDescription = "Filter by group ID"

	SinceDescription = "Filter executions created after this time (ISO 8601 format)"

	FilenameDescription = "The name of the artifact file to retrieve"
)
