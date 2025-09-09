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

	PageDescription = "Page number for pagination (default: 0)"

	PageSizeDescription = "Number of items to return per page (default: 10, max: 100)"

	TextSearchDescription = `Text search filter for names or descriptions. Can use space-separated words 
to find items containing all terms`

	SelectorDescription = `Filter by labels using key=value format. For single label use 'key=value', for multiple labels use comma-separated format 'key1=value1,key2=value2'. For example: 'tool=cypress' or 'tool=cypress,env=prod'`

	StatusDescription = `Filter by execution status. Available statuses: 'queued', 'running', 'passed', 
'failed', 'skipped', 'aborted', 'timeout', 'paused'`

	ResourceGroupDescription = "Filter by resource group using the group slug (e.g., 'demo-resource-group', 'accounting-tests'). Use the list_resource_groups tool to discover available groups"

	SinceDescription = "Filter executions created after this time (ISO 8601 format)"

	FilenameDescription = "The name of the artifact file to retrieve"

	// Workflow tool descriptions
	ListWorkflowsDescription         = "List Testkube workflows with optional filtering by resource group, selector, status, and other criteria. Returns workflow names (which are also the workflow IDs), descriptions, and execution status."
	CreateWorkflowDescription        = "Create a new TestWorkflow directly in Testkube from a YAML definition. Use this tool to deploy workflows to the Testkube platform. The workflow will be immediately available for execution after creation."
	GetWorkflowDefinitionDescription = "Get the YAML definition of a specific Testkube workflow. Returns the complete workflow specification including all steps, configuration schema, and metadata."
	GetWorkflowDescription           = "Retrieve detailed workflow information including execution history, health metrics, and current status. Returns JSON format with comprehensive workflow metadata."
	RunWorkflowDescription           = "Run a TestWorkflow with optional configuration parameters. If the workflow requires config parameters, use the get_workflow_definition tool first to examine the spec.config section to see what parameters are available."
	UpdateWorkflowDescription        = `Update an existing TestWorkflow in Testkube with a new YAML definition. This tool allows you to modify workflow steps, configuration, and metadata. The workflow will be updated immediately and available for execution with the new configuration.`

	// Execution tool descriptions
	FetchExecutionLogsDescription     = "Retrieves the full logs of a test workflow execution for debugging and analysis."
	ListExecutionsDescription         = "List executions for a specific test workflow with filtering and pagination options. Returns execution summaries with status, timing, and results."
	GetExecutionInfoDescription       = "Get detailed information about a specific test workflow execution, including status, timing, results, and configuration."
	LookupExecutionIdDescription      = "Resolves an execution name to its corresponding execution ID. Use this tool when you have an execution name (e.g., 'my-workflow-123', 'my-test-987-1') but need the execution ID. Many other tools require execution IDs (MongoDB format) rather than names."
	AbortWorkflowExecutionDescription = "Abort a running test workflow execution. This will stop the execution and mark it as aborted. Use this tool to cancel long-running or stuck workflow executions."

	// Artifact tool descriptions
	ListArtifactsDescription = "Retrieves all artifacts generated during a workflow execution. Use this tool to discover available outputs, reports, logs, or other files produced by test runs. These artifacts provide valuable context for understanding test results, accessing detailed reports, or examining generated data. The response includes artifact names, sizes, and their current status."
	ReadArtifactDescription  = "Retrieves the content of a specific artifact from a workflow execution. This tool fetches up to 100 lines of text content from the requested file."

	// Other tool descriptions
	BuildDashboardUrlDescription  = "Build dashboard URLs for Testkube workflows and executions."
	ListLabelsDescription         = "Retrieve all available labels and their values from workflows in the current Testkube environment. Returns a map where each key is a label name and the value is an array of all possible values for that label. This is useful for discovering what labels exist and what values you can filter by when using selectors in other tools."
	ListResourceGroupsDescription = "Retrieve all available resource groups from the current Testkube environment. Returns a list of resource groups with their IDs, slugs, names, descriptions, and metadata. This is useful for discovering what resource groups exist and what slugs you can use when filtering by resource groups in other tools."
)
