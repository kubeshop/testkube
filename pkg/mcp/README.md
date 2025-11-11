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
	GetExecutionLogs(ctx context.Context, executionId string) (string, error)
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

The MCP server exposes 20 tools organized into five categories:

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

#### Execution Tools (7 tools)

- `fetch_execution_logs` - Fetch logs for specific execution
- `list_executions` - List executions with optional workflow name and filtering
- `lookup_execution_id` - Look up execution ID by execution name
- `get_execution_info` - Get detailed execution information
- `get_workflow_execution_metrics` - Fetch metrics for specific execution
- `wait_for_executions` - Poll multiple executions until completion (5s interval)
- `abort_workflow_execution` - Abort running workflow execution

#### Artifact Tools (2 tools)

- `list_artifacts` - List artifacts for an execution
- `read_artifact` - Read artifact content (handles both direct content and S3 URLs)

#### Metadata Tools (3 tools)

- `list_labels` - List all labels in the environment
- `list_resource_groups` - List resource groups in the organization
- `list_agents` - List agents with filtering (type, capability, pagination)

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
