package tools

const (
	WorkflowNameDescription = `The name of the workflow. Workflow names are lowercase alphanumeric with dashes 
(e.g., 'my-workflow', 'api-tests'). This uniquely identifies a TestWorkflow within the organization.`

	ExecutionIdDescription = `The unique execution ID in MongoDB ObjectID format (24 hex chars, e.g., '67d2cdbc351aecb2720afdf2').
Use lookup_execution_id if you only have an execution name.`

	ExecutionNameDescription = `The name of the execution (e.g., 'my-workflow-123'). Execution names follow
the pattern of workflow name plus a numeric suffix. Use lookup_execution_id to get the ID from a name.`

	PageDescription = "Page number for pagination (default: 0)"

	PageSizeDescription = "Number of items to return per page (default: 10, max: 100)"

	TextSearchDescription = `Text search filter for names or descriptions. Can use space-separated words
to find items containing all terms`

	SelectorDescription = `Filter workflows by label using key=value format. For single label use 'key=value',
for multiple labels use comma-separated format 'key1=value1,key2=value2' (e.g., 'tool=cypress,env=prod').
Note: filters workflow-level labels, not execution tags — use tagSelector for execution tags.`

	TagSelectorDescription = `Filter executions by tag using key=value format. For single tag use 'key=value',
for multiple tags use comma-separated format 'key1=value1,key2=value2' (e.g., 'type=suite,env=prod').
Note: filters execution-level tags (set via update_execution_tags), not workflow labels — use selector for workflow labels.`

	StatusDescription = `Filter by execution status. Available statuses: 'queued', 'running', 'passed', 
'failed', 'skipped', 'aborted', 'timeout', 'paused'`

	ResourceGroupDescription = "Filter by resource group using the group slug (e.g., 'demo-resource-group', 'accounting-tests'). Use the list_resource_groups tool to discover available groups"

	SinceDescription = "Filter executions created after this time (ISO 8601 format)"

	StartDateDescription = `Filter items on or after this time. Accepts a date (YYYY-MM-DD, e.g., '2024-01-15')
or an RFC 3339 timestamp (e.g., '2024-01-15T13:00:00Z').`

	EndDateDescription = `Filter items on or before this time. Accepts a date (YYYY-MM-DD, e.g., '2024-01-31')
or an RFC 3339 timestamp (e.g., '2024-01-31T16:00:00Z'). Combine with startDate for date ranges.`

	FilenameDescription = "The name of the artifact file to retrieve"

	// Workflow tool descriptions
	ListWorkflowsDescription               = "List Testkube workflows with optional filtering by resource group, selector, status, and text search. Returns workflow names, descriptions, and execution status."
	CreateWorkflowDescription              = "Create a new TestWorkflow from a YAML definition. The workflow is immediately available for execution after creation."
	GetWorkflowDefinitionDescription       = "Get the YAML definition of a specific Testkube workflow. Returns the complete specification including steps, configuration schema, and metadata."
	GetWorkflowDescription                 = "Get detailed workflow information including execution history, health metrics, and current status."
	GetWorkflowMetricsDescription          = "Get execution metrics for a workflow: execution statistics, health scores, pass rates, and performance data. Use to analyze workflow reliability over time."
	GetWorkflowExecutionMetricsDescription = "Get raw resource metrics (CPU, memory, disk, network) for a single workflow execution as time-series data. Use for deep-dive debugging of a specific run. Requires workflowName and executionId."
	GetWorkflowResourceHistoryDescription  = "Analyze resource consumption patterns (CPU, memory, disk, network) across recent executions of a workflow. Computes cross-execution statistics (mean, min, max, stdDev), detects trends, and identifies outliers. Use to investigate growing resource usage or find abnormal runs. Requires workflowName."
	RunWorkflowDescription                 = "Run a TestWorkflow with optional configuration parameters and agent targeting. Use get_workflow_definition first to discover available config parameters. Use list_agents to discover available target agents."
	UpdateWorkflowDescription              = "Update an existing TestWorkflow with a new YAML definition. The workflow is updated immediately and available for execution with the new configuration."

	// Execution tool descriptions
	FetchExecutionLogsDescription           = "Retrieve logs from a test workflow execution. Default returns last 100 lines. Use grep to search the full log (capped at 100 matches). Always paginate in 100-line chunks. For parallel workflows with workers, call get_execution_info first to get valid worker refs (do not use step refs)."
	ListExecutionsDescription               = "List test workflow executions with filtering by workflow, status, date range, labels, tags, or text search. Returns execution summaries including status, duration, and metadata. Use to discover recent runs or find specific executions."
	GetExecutionInfoDescription             = "Get detailed information about a specific workflow execution: status, timing, results, configuration, and worker instances. Requires executionId. workflowName is optional for disambiguation."
	GetExecutionInfoWorkflowNameDescription = "Optional workflow name for scoping an execution name lookup. Safe to omit when you have an execution ID."
	LookupExecutionIdDescription            = "Resolve an execution name (e.g., 'my-workflow-123') to its execution ID. Use when you have an execution name but need the ID for other tools."
	WaitForExecutionsDescription            = "Wait for a list of workflow executions to complete. Returns the final status of all executions. Use for synchronizing dependent workflows."
	AbortWorkflowExecutionDescription       = "Abort a running workflow execution. Stops the execution and marks it as aborted. Use for cancelling long-running or stuck executions."
	UpdateExecutionTagsDescription          = "Update tags on a workflow execution. Uses replace semantics: provided tags completely replace existing tags. Send empty map {} to clear all tags. Tags are key-value pairs for categorization and filtering."

	// Additional parameter descriptions
	ExecutionIdsDescription   = "Comma-separated list of execution IDs to wait for (e.g., 'exec1,exec2,exec3')."
	TimeoutMinutesDescription = "Maximum time to wait in minutes before timing out (default: 30). Set to 0 for no timeout."

	// Artifact tool descriptions
	ListArtifactsDescription = "List all artifacts (files, reports, logs) generated by a workflow execution. Returns artifact names, sizes, and status. Use to discover available outputs before reading specific artifacts."
	ReadArtifactDescription  = "Read content from an artifact file produced by a workflow execution. Default returns first 100 lines (max 200 per request). Always paginate in 100-200 line chunks. For binary artifacts, returns a summary instead of content."

	// Other tool descriptions
	BuildDashboardUrlDescription  = "Build dashboard URLs for Testkube workflows and executions. Supports deep linking to a specific step in the execution log view via stepRef."
	ListLabelsDescription         = "List all workflow labels and their values in the environment. Use to discover available filters for selector parameters in other tools."
	ListResourceGroupsDescription = "List available resource groups in the organization. Use to discover group slugs for filtering workflows and executions."
	ListAgentsDescription         = "List available agents in the organization for workflow execution targeting. Returns agent names, types, capabilities, labels, and status. Use before run_workflow to discover target agents."

	// Query tool descriptions
	QueryWorkflowsDescription = `Query workflow definitions in bulk using JSONPath.
Fetches workflow YAML and extracts fields matching the expression.
Use to find workflows by configuration patterns, image references, or step structure across all workflows.`

	QueryExecutionsDescription = `Query execution records across multiple workflows using JSONPath.
Fetches execution data and extracts fields matching the expression.
Use for cross-workflow analysis: find all failed executions, compare durations, or extract specific fields in bulk.`

	// WorkflowTemplate tool descriptions
	TemplateNameDescription = `The name of the workflow template. Template names are lowercase alphanumeric with dashes 
(e.g., 'my-template', 'official--k6--v1'). This uniquely identifies a TestWorkflowTemplate within the environment.`

	ListWorkflowTemplatesDescription         = "List all TestWorkflowTemplates with optional label filtering. Returns template names, descriptions, and labels."
	GetWorkflowTemplateDefinitionDescription = "Get the YAML definition of a specific TestWorkflowTemplate. Returns the complete template specification including steps, config schema, and metadata."
	CreateWorkflowTemplateDescription        = "Create a new TestWorkflowTemplate from a YAML definition. The template is immediately available for use by workflows."
	UpdateWorkflowTemplateDescription        = "Update an existing TestWorkflowTemplate with a new YAML definition. Workflows using the template pick up the changes."

	// Schema tool descriptions
	GetWorkflowSchemaDescription  = "Get the YAML schema for TestWorkflow definitions. Returns all available fields, their types, and descriptions. Use to understand workflow structure when creating or querying workflows."
	GetExecutionSchemaDescription = "Get the YAML schema for TestWorkflowExecution data. Returns all available fields, their types, and descriptions. Use to understand execution data structure when analyzing results."

	// Insight (ingested metrics) tool descriptions
	ListInsightSeriesDescription = `Discover the granular insight metric series ingested from test/performance reports.
Each series is one metric for one workflow (and optional identity, e.g. a specific k6 request name or JUnit test case), identified by a seriesId.
Series are parsed from k6, jmeter, artillery, junit and influx reports. Use this to find which metrics exist and to get the seriesId and metricKey to feed into get_insight_metric_series.
Scoped to the current environment.`

	ListInsightMetricKeysDescription = `List the distinct granular insight metric keys available (e.g. 'http_req_duration_p95_ms', 'response_time_percentile_2_ms', 'test_duration_ms').
A lightweight vocabulary for discovery; use list_insight_series when you also need seriesId, source, workflow or identity. Scoped to the current environment.`

	GetInsightMetricSeriesDescription = `Query a granular insight metric as a time series (values and trends over time) for the current environment.
Provide either a metricKey via 'measure' (values are aggregated across all matching series) or one or more 'seriesId' values (comma-separated). If both are set, seriesId wins.
Discover metric keys with list_insight_metric_keys and series/seriesIds with list_insight_series.
Besides granular metric keys, 'measure' also accepts the canonical cross-tool measures 'latency_p95_ms', 'throughput_rps' and 'errors_rate' (these do NOT appear in the series catalog).
Use 'segment' to break the series down by workflow, status, or an identity field key. Data defaults to the last 7 days unless startDate/endDate are given.`

	ListInsightExecutionsDescription = `List the workflow executions that produced a given insight metric, most recent first.
Use after get_insight_metric_series to drill from a metric value/trend down to the concrete executions behind it, then pivot to get_execution_info or fetch_execution_logs.
Filter with the same 'measure'/identity/workflow/status/tag/date filters as get_insight_metric_series. Scoped to the current environment.`

	// Insight tool parameter descriptions
	InsightSourceDescription     = "Filter by the report source that produced the series: 'k6', 'jmeter', 'artillery', 'junit', 'influx', or 'canonical'."
	InsightMetricKeyDescription  = "Filter by an exact metric key (e.g. 'http_req_duration_p95_ms', 'test_duration_ms'). Discover valid keys with list_insight_metric_keys."
	InsightQueryDescription      = "Free-text search across series metadata (metric key, workflow, identity)."
	InsightMeasureDescription    = `The metric to query. A granular metric key (e.g. 'http_req_duration_p95_ms') or a canonical cross-tool measure ('latency_p95_ms', 'throughput_rps', 'errors_rate'). Discover granular keys with list_insight_metric_keys.`
	InsightSeriesIdDescription   = "Comma-separated granular insight series IDs to include (from list_insight_series). When set, takes precedence over 'measure'."
	InsightAggregateDescription  = "How to aggregate values within each time bucket: 'avg' (default), 'sum', 'min', 'max', or 'count'."
	InsightSegmentDescription    = "Break the series down by a property: 'workflow', 'status', or any stable identity field key (e.g. 'testcase', 'scenario', 'route')."
	InsightMaxSamplesDescription = "Maximum number of time-series points to return per segment (default: 50). Increase for finer detail, decrease for a more compact response."
	InsightWorkflowDescription   = "Filter to a single workflow by name."

	InsightIdentityFiltersDescription = `JSON object of granular insight series identity filters. Keys are identity field names and values are arrays of strings. ` +
		`Plain strings match exactly (case-insensitive); strings prefixed with "~" match partially via regex. ` +
		`Multiple identity keys are combined with AND; multiple values for the same key are combined with OR. ` +
		`Example: {"testcase":["login"],"route":["~^/api/"]}.`
	InsightTagFilterDescription = `Filter executions by tag. Tag values may contain commas, so multiple predicates must be passed as a JSON array, e.g. ["release=2026.07,hotfix","bug"]. ` +
		`A plain (non-JSON) string is treated as a single predicate verbatim. Each predicate supports key=value (exact), key=~pattern (regex), or a bare key (existence).`
)
