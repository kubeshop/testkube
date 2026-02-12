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
	ListWorkflowsDescription               = "List Testkube workflows with optional filtering by resource group, selector, status, and other criteria. Returns workflow names (which are also the workflow IDs), descriptions, and execution status."
	CreateWorkflowDescription              = "Create a new TestWorkflow directly in Testkube from a YAML definition. Use this tool to deploy workflows to the Testkube platform. The workflow will be immediately available for execution after creation."
	GetWorkflowDefinitionDescription       = "Get the YAML definition of a specific Testkube workflow. Returns the complete workflow specification including all steps, configuration schema, and metadata."
	GetWorkflowDescription                 = "Retrieve detailed workflow information including execution history, health metrics, and current status. Returns JSON format with comprehensive workflow metadata."
	GetWorkflowMetricsDescription          = "Get metrics of test workflow executions including execution statistics, health scores, pass rates, and performance data. Returns comprehensive metrics data for analyzing workflow performance and reliability."
	GetWorkflowExecutionMetricsDescription = "Get detailed resource consumption metrics for a SINGLE workflow execution. Returns raw time-series data (CPU, memory, disk, network samples over time) for deep analysis of one specific run. Use this when you need granular metrics or charts for debugging a particular execution. Requires both workflowName and executionId."
	GetWorkflowResourceHistoryDescription  = "Analyze resource consumption PATTERNS across multiple executions of a workflow. Fetches the last N executions (default 50) and computes cross-execution statistics: mean/min/max/stdDev, trend detection (increasing/decreasing/stable), and outlier identification (z-score > 2). Use this to answer questions like 'is memory usage growing over time?' or 'which runs had abnormal CPU usage?'. Only requires workflowName."
	RunWorkflowDescription                 = "Run a TestWorkflow with optional configuration parameters and target specification. If the workflow requires config parameters, use the get_workflow_definition tool first to examine the spec.config section to see what parameters are available. The target parameter supports multiple formats: 1) {\"name\": \"agent-name\"} to target a specific runner by name, 2) {\"labels\": {\"env\": \"prod\", \"type\": \"runner\"}} to target runners by labels, 3) Standard ExecutionTarget format with match/not/replicate fields."
	UpdateWorkflowDescription              = `Update an existing TestWorkflow in Testkube with a new YAML definition. This tool allows you to modify workflow steps, configuration, and metadata. The workflow will be updated immediately and available for execution with the new configuration.`

	// Execution tool descriptions
	FetchExecutionLogsDescription     = "Retrieves the full logs of a test workflow execution for debugging and analysis."
	ListExecutionsDescription         = "List executions with filtering and pagination options. Optionally filter by workflow name, status, or text search. Returns execution summaries with status, timing, and results."
	GetExecutionInfoDescription       = "Get detailed information about a specific test workflow execution, including status, timing, results, and configuration."
	LookupExecutionIdDescription      = "Resolves an execution name to its corresponding execution ID. Use this tool when you have an execution name (e.g., 'my-workflow-123', 'my-test-987-1') but need the execution ID. Many other tools require execution IDs (MongoDB format) rather than names."
	WaitForExecutionsDescription      = "Wait for a list of workflow executions to complete (pass, fail, or timeout). Returns the final status of all executions. Useful for synchronizing multiple test runs or waiting for dependent workflows to finish."
	AbortWorkflowExecutionDescription = "Abort a running test workflow execution. This will stop the execution and mark it as aborted. Use this tool to cancel long-running or stuck workflow executions."

	// Additional parameter descriptions
	ExecutionIdsDescription   = "Comma-separated list of execution IDs to wait for (e.g., 'exec1,exec2,exec3')."
	TimeoutMinutesDescription = "Maximum time to wait in minutes before timing out (default: 30). Set to 0 for no timeout."

	// Artifact tool descriptions
	ListArtifactsDescription = "Retrieves all artifacts generated during a workflow execution. Use this tool to discover available outputs, reports, logs, or other files produced by test runs. These artifacts provide valuable context for understanding test results, accessing detailed reports, or examining generated data. The response includes artifact names, sizes, and their current status."
	ReadArtifactDescription  = "Retrieves the content of a specific artifact from a workflow execution. This tool fetches up to 100 lines of text content from the requested file."

	// Other tool descriptions
	BuildDashboardUrlDescription  = "Build dashboard URLs for Testkube workflows and executions."
	ListLabelsDescription         = "Retrieve all available labels and their values from workflows in the current Testkube environment. Returns a map where each key is a label name and the value is an array of all possible values for that label. This is useful for discovering what labels exist and what values you can filter by when using selectors in other tools."
	ListResourceGroupsDescription = "Retrieve all available resource groups from the current Testkube environment. Returns a list of resource groups with their IDs, slugs, names, descriptions, and metadata. This is useful for discovering what resource groups exist and what slugs you can use when filtering by resource groups in other tools."
	ListAgentsDescription         = "Retrieve all available agents from the current Testkube organization that can be used when specifying target agent(s) in run_workflow. Returns a list of agents with their IDs, names, types, capabilities, labels, and environment information. This is useful for discovering what agents are available for workflow execution targeting."

	// Query tool descriptions
	QueryWorkflowsDescription = `Query multiple workflow definitions using JSONPath expressions.
Fetches workflow YAML definitions and extracts data matching the path.

Supported JSONPath syntax:
  $                   - Root element (the workflow)
  $.spec.steps        - Direct path to steps array
  $.spec.steps[0]     - First step
  $.spec.steps[*]     - All steps
  $..image            - All 'image' fields anywhere (recursive)
  $[?(@.name=='x')]   - Filter by field value

Parameters:
- expression: The JSONPath expression to apply (required)
- selector: Filter workflows by labels (e.g., 'tool=cypress,env=prod')
- resourceGroup: Filter by resource group slug
- limit: Maximum workflows to fetch (default 50, max 100)
- aggregate: If true, combines all workflows into an array and applies expression once; if false, applies expression to each workflow separately

Returns: Map of workflow name → extracted values. Missing paths return empty arrays, not errors.`

	QueryExecutionsDescription = `Query multiple execution records using JSONPath expressions.
Fetches execution JSON data and extracts data matching the path.

Supported JSONPath syntax:
  $                   - Root element (the execution)
  $.result.status     - Direct path to status
  $.result.steps.*    - All step results (steps is a map, not array)
  $..duration         - All duration fields (recursive)
  $[?(@.status=='failed')] - Filter by status

Parameters:
- expression: The JSONPath expression to apply (required)
- workflowName: Filter executions by workflow name
- status: Filter by status (passed/failed/running/aborted)
- limit: Maximum executions to fetch (default 50, max 100)
- aggregate: If true, combines all executions into an array and applies expression once; if false, applies expression to each execution separately

Returns: Map of execution ID → extracted values. Missing paths return empty arrays, not errors.`

	// WorkflowTemplate tool descriptions
	TemplateNameDescription = `The name of the workflow template. Template names are lowercase alphanumeric with dashes 
(e.g., 'my-template', 'official--k6--v1'). This uniquely identifies a TestWorkflowTemplate within the environment.`

	ListWorkflowTemplatesDescription         = "List all TestWorkflowTemplates in the current Testkube environment with optional label filtering. Returns template names, descriptions, and labels."
	GetWorkflowTemplateDefinitionDescription = "Get the YAML definition of a specific TestWorkflowTemplate. Returns the complete template specification including all steps, configuration schema, and metadata."
	CreateWorkflowTemplateDescription        = "Create a new TestWorkflowTemplate in Testkube from a YAML definition. The template will be immediately available for use by workflows after creation."
	UpdateWorkflowTemplateDescription        = "Update an existing TestWorkflowTemplate in Testkube with a new YAML definition. The template will be updated immediately and workflows using it will pick up the changes."

	// Schema tool descriptions
	GetWorkflowSchemaDescription  = "Get the YAML schema for TestWorkflow definitions. Returns all available fields, their types, and descriptions. Use this to understand workflow structure when creating, updating, or querying workflows."
	GetExecutionSchemaDescription = "Get the YAML schema for TestWorkflowExecution data. Returns all available fields, their types, and descriptions. Use this to understand execution data structure when analyzing results or querying executions."
)
