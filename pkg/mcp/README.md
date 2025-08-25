# Testkube MCP (Model Context Protocol) Integration

This package provides MCP integration for Testkube, enabling AI assistants to interact with Testkube workflows, executions, and artifacts through the [Model Context Protocol](https://modelcontextprotocol.io).

## Overview

The MCP server is exposed via the `testkube mcp serve` CLI command, which leverages the existing OAuth authentication flow for security purposes. The server requires you to be logged in via `testkube login` so it can access your specific Testkube account, permissions, and context (organization and environment). This provides seamless setup since the CLI already has all the necessary configuration after login.

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

- **HTTP Client** (default): Makes REST API calls to Testkube endpoints
- **Direct Repository Access** (future): For control plane integration with direct database access

This flexibility allows the same MCP tools to work in different deployment scenarios without code changes.

## Usage

### Starting the MCP Server

```bash
# Build the CLI
make build-kubectl-testkube-cli
# This also deletes the previously built cli
make rebuild-kubectl-testkube-cli

# Start the MCP server
./bin/app/kubectl-testkube mcp serve

# Start with debug output
./bin/app/kubectl-testkube mcp serve --debug

# Use --verbose if you need to check what context is used, but this will log things to stdout
./bin/app/kubectl-testkube mcp serve --verbose
```

### Development and Testing

Use the MCP inspector to test tools interactively:

```bash
npx @modelcontextprotocol/inspector ./bin/app/kubectl-testkube mcp serve --debug
```

The debug mode enables detailed request/response logging for the API client, making it easier to troubleshoot integration issues.

### Example MCP configuration for VSCode

```
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
