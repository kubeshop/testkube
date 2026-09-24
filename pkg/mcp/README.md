````markdown
# Testkube MCP (Model Context Protocol) Integration

This package provides MCP integration for Testkube, enabling AI assistants to interact with Testkube workflows, executions, and artifacts through the [Model Context Protocol](https://modelcontextprotocol.io).

## Overview

The MCP integration supports two deployment modes:

1. **CLI Mode** (`testkube mcp serve`): Runs locally with stdio or SHTTP transport, authenticating via OAuth or API keys
2. **Control Plane Mode**: Embedded HTTP endpoint at `/organizations/{orgId}/environments/{envId}/mcp` with SSE transport

The CLI mode requires authentication via `testkube login` or `testkube set context` with an API key. The control plane mode uses standard bearer token authentication with per-environment feature flags for access control.

This implementation uses the [mcp-go](https://github.com/mark3labs/mcp-go) library, chosen for its proven usage in other projects like [GitHub's MCP server](https://github.com/github/github-mcp-server). The tool design patterns and helper functions draw inspiration from GitHub's implementation while being adapted for Testkube's specific needs.

## Architecture

### Tool Design Pattern

The MCP tools follow a consistent, interface-based design:

```go
// Small focused interface, include strictly methods called by the tool handler
type ExecutionLogger interface {
  // Client receive the context and any number of additional parameters methods, must return (string, error)
	GetExecutionLogs(ctx context.Context, executionId string, params ExecutionLogParams) (string, error)
}

func FetchExecutionLogs(client ExecutionLogger) (tool mcp.Tool, handler server.ToolHandlerFunc)
```

Each tool function:

- Receives a small, focused interface (e.g., `ExecutionLogger`, `ArtifactLister`, `WorkflowRunner`)
- Returns an `mcp.Tool` definition and a `ToolHandlerFunc` from the mcp-go library
- Maintains clear separation of concerns and testability

### Client Abstraction

The package uses an interface-based client design that supports multiple implementations:

- **APIClient** (CLI mode): Makes REST API calls to control plane endpoints via HTTP
- **HandlerClient** (control plane mode): Invokes API handlers directly in-process for low-latency operation

This flexibility allows the same MCP tools to work in different deployment scenarios without code changes. The control plane can implement its own client that calls handlers directly while the CLI uses HTTP transport.

### Available Tools

The MCP server exposes up to 43 tools organized into the categories below. The two
Query tools register conditionally: with the default `APIClient`, they are added only
when the control plane advertises the required endpoints (unless `SkipEndpointChecks`
is set); other client implementations register them unconditionally. The Insight
tools register unconditionally and require a control plane that serves the
`/insights/*` endpoints; against an older control plane they appear in the tool list
but return an error when called.

#### Dashboard Tools (1 tool)

- `build_dashboard_url` - Generate dashboard URLs for workflows and executions

#### Workflow Tools (7 tools)

- `list_workflows` - List workflows with filtering (selector, text search, pagination)
- `get_workflow` - Retrieve specific workflow by name
- `get_workflow_definition` - Return formatted workflow definition (same as get_workflow)
- `get_workflow_metrics` - Fetch workflow metrics
- `create_workflow` - Create new workflow from YAML/JSON definition
- `update_workflow` - Update existing workflow
- `run_workflow` - Execute workflow with config and target parameters

#### Workflow Template Tools (4 tools)

- `list_workflowtemplates` - List workflow templates with optional label filtering
- `get_workflowtemplate_definition` - Get the YAML definition of a specific template
- `create_workflowtemplate` - Create a new template from a YAML definition
- `update_workflowtemplate` - Update an existing template with a new YAML definition

#### Query Tools (2 tools, registered conditionally)

- `query_workflows` - Bulk-query workflow definitions using JSONPath
- `query_executions` - Bulk-query execution records across workflows using JSONPath

#### Schema Tools (2 tools)

- `get_workflow_schema` - Get the YAML schema for TestWorkflow definitions
- `get_execution_schema` - Get the YAML schema for TestWorkflowExecution data

#### Execution Tools (9 tools)

- `fetch_execution_logs` - Fetch logs for specific execution
- `list_executions` - List executions with optional workflow name and filtering
- `lookup_execution_id` - Look up execution ID by execution name
- `get_execution_info` - Get detailed execution information
- `get_workflow_execution_metrics` - Fetch metrics for specific execution
- `get_workflow_resource_history` - Analyze resource consumption (CPU, memory, disk, network) across recent executions of a workflow
- `wait_for_executions` - Poll multiple executions until completion (5s interval)
- `abort_workflow_execution` - Abort running workflow execution
- `update_execution_tags` - Update tags on an execution (replace semantics)

#### Artifact Tools (2 tools)

- `list_artifacts` - List artifacts for an execution
- `read_artifact` - Read artifact content (handles both direct content and S3 URLs)

#### Metadata Tools (3 tools)

- `list_labels` - List all labels in the environment
- `list_resource_groups` - List resource groups in the organization
- `list_agents` - List agents with filtering (type, capability, pagination)

#### Insight Tools (4 tools)

Expose the ingested granular insight series (performance/test metrics parsed from k6, JMeter, Artillery, JUnit, and Influx reports, plus cross-tool canonical metrics). All insight tools are scoped to the current environment automatically.

- `list_insight_series` - Discover the granular insight metric series ingested from test/performance reports
- `list_insight_metric_keys` - List the distinct insight metric keys available (lightweight vocabulary for discovery)
- `get_insight_metric_series` - Query a granular insight metric as a time series (values and trends over time)
- `list_insight_executions` - List the workflow executions that produced a given insight metric

#### Insights Board Tools (9 tools)

Manage Insights boards: saved, organization-wide dashboards made of reports (charts). Boards are organization-scoped, not environment-scoped; a report narrows itself to environments through its own `environment` filter. The Control Plane serves boards only to signed-in users, so these tools require a user session (`testkube login`, or the hosted endpoint with a user account) and return an actionable error for API tokens — the `APIClient` refuses a `tkcapi_` token before sending anything. They register unconditionally: the board routes do not answer HEAD, so they cannot be probed.

- `list_boards` - List the boards visible to you (shared and private), with name, visibility and pinned filters
- `get_board` - Get a board with its reports in dashboard order and its layout
- `create_board` - Create an empty board (shared by default)
- `update_board` - Change a board's name, description, slug, visibility or layout
- `add_board_report` - Add a `pass-fail`, `executions`, `workflows` or `time-series` report
- `update_board_report` - Change a report's name, description, kind or params (merged by default)
- `remove_board_report` - Remove a report (destructive)
- `delete_board` - Delete a board and its reports (destructive)
- `render_board` - Run each report's query and return its numbers; `scope=environment` limits every report to the current environment, and `timeZone` (IANA, default UTC) anchors relative ranges at that zone's midnight, as the dashboard anchors them at the viewer's local midnight.

Report params have no backend schema: the Control Plane stores them opaquely and only the dashboard interprets them. `pkg/mcp/boards` holds the port of the dashboard's rules — param validation and defaults, the report-to-query translation, and the layout — and both the `APIClient` and the Control Plane's `HandlerClient` must use it rather than reimplement it. The translation is pinned by `pkg/mcp/boards/testdata/translation_cases.json`. Two Control Plane behaviors the tools work around: every board update clears the description unless it is sent, so each write reads the board and resends it; and deleting a report leaves its layout cell behind, so `remove_board_report` sends the recomputed layout with the delete. Because those values come from a read, every update also sends `expectedVersion`, the board's version as read: the Control Plane refuses the write with 409 if the board changed in between, and the tool reads the board again and rebuilds the write from it (up to three attempts) instead of overwriting the newer edit. `delete_board` resends nothing it read, so it is not conditional: it deletes by the ID it resolved, and the Control Plane checks delete rights against the board as it is then.

**Note for maintainers:** When adding new tools to `pkg/mcp/tools/`, ensure that:

1. The tool follows the interface-based design pattern (see existing tools for examples)
2. The tool is registered in both:
   - `pkg/mcp/server.go` (`NewMCPServer` function) for CLI mode
   - Control plane's `mcp_handler.go` (`createMCPServer` function) for embedded mode
3. If the tool requires a new client method, implement it in both:
   - `pkg/mcp/api.go` (`APIClient`) for HTTP-based CLI access
   - Control plane's `mcp_client.go` (`HandlerClient`) for direct handler invocation

### Middleware and Debug Support

The MCP server includes middleware for:

- **Debug Middleware**: Adds detailed request/response metadata when debug mode is enabled (via `--debug` flag or `?debug=true` query param)
- **Telemetry Middleware**: Tracks tool invocations for usage analytics (when telemetry is enabled)

Debug mode attaches metadata to tool responses under `_meta.debug`, showing the data source (HTTP or handler), request details, status codes, and headers.

### Docker Image

There is a Docker image available for the MCP-Server on DockerHub - https://hub.docker.com/repository/docker/testkube/mcp-server - see build and usage instructions at [/build/mcp-server/README.md](../../build/mcp-server/README.md).

## Usage

See extensive docs at https://docs.testkube.io/articles/mcp-overview.

### Starting the MCP Server

#### Stdio Transport (Default)

```bash
# Build the CLI
make build-kubectl-testkube-cli

# This also deletes the previously built cli
make rebuild-kubectl-testkube-cli

# Start the MCP server with stdio transport (default)
./bin/app/kubectl-testkube mcp serve

# Start with debug output
./bin/app/kubectl-testkube mcp serve --debug

# Use --verbose if you need to check what context is used, but this will log things to stdout
./bin/app/kubectl-testkube mcp serve --verbose
```

#### Streamable HTTP (SHTTP) Transport

```bash
# Start MCP server with SHTTP transport on localhost:8080
./bin/app/kubectl-testkube mcp serve --transport=shttp

# Start SHTTP server on custom host and port
./bin/app/kubectl-testkube mcp serve --transport=shttp --shttp-host=0.0.0.0 --shttp-port=9090

# Start SHTTP server with TLS
./bin/app/kubectl-testkube mcp serve --transport=shttp --shttp-tls --shttp-cert-file=cert.pem --shttp-key-file=key.pem

# Start SHTTP server with debug output
./bin/app/kubectl-testkube mcp serve --transport=shttp --debug
```

### Development and Testing

Use the MCP inspector to test tools interactively:

```bash
npx @modelcontextprotocol/inspector ./bin/app/kubectl-testkube mcp serve --debug
```

The debug mode enables detailed request/response logging for the API client, making it easier to troubleshoot integration issues.

### Example MCP configuration for VSCode

#### Stdio Configuration

```json
{
  "servers": {
    "testkube": {
      "command": "/path/to/your/testkube/bin/app/kubectl-testkube",
      "args": ["mcp", "serve", "--debug"],
      "type": "stdio"
    }
  }
}
```

#### SHTTP Configuration

```json
{
  "servers": {
    "testkube": {
      "command": "/path/to/your/testkube/bin/app/kubectl-testkube",
      "args": ["mcp", "serve", "--transport=shttp", "--shttp-host=localhost", "--shttp-port=8080"],
      "type": "shttp"
    }
  }
}
```
````
