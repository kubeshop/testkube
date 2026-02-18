# MCP Tool Implementation Reference

Complete example showing the `list_labels` tool across all files.

---

## 1. Tool definition — `pkg/mcp/tools/labels.go`

```go
package tools

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

type LabelsLister interface {
    ListLabels(ctx context.Context) (string, error)
}

func ListLabels(client LabelsLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
    tool = mcp.NewTool("list_labels",
        mcp.WithDescription(ListLabelsDescription),
    )

    handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        result, err := client.ListLabels(ctx)
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
        return mcp.NewToolResultText(result), nil
    }

    return tool, handler
}
```

---

## 2. Client interface — `pkg/mcp/client.go`

```go
type Client interface {
    tools.ArtifactLister
    tools.ArtifactReader
    tools.ExecutionLogger
    // ...
    tools.LabelsLister          // <-- included here
    tools.ResourceGroupsLister
    // ...
}
```

---

## 3. CLI client — `pkg/mcp/api.go`

The `APIClient` calls the agent HTTP API:

```go
func (c *APIClient) ListLabels(ctx context.Context) (string, error) {
    return c.get(ctx, "/agent/labels")
}
```

---

## 4. Registration in testkube — `pkg/mcp/server.go`

```go
func NewMCPServer(cfg MCPServerConfig, client Client) (*server.MCPServer, error) {
    // ...
    mcpServer.AddTool(tools.ListLabels(client))
    // ...
}
```

---

## 5. Registration in cloud-api — `internal/server/mcp_handler.go`

```go
func (a *APIService) createMCPServer(client *HandlerClient, debug bool) (*server.MCPServer, error) {
    // ...
    mcpServer.AddTool(mcptools.ListLabels(client))
    // ...
}
```

---

## 6. Cloud-side client — `internal/server/mcp_client.go`

The `HandlerClient` invokes API handlers in-process (no network hop):

```go
func (c *HandlerClient) ListLabels(ctx context.Context) (string, error) {
    req := c.newRequest(ctx, http.MethodGet, "/agent/labels", nil)
    w := httptest.NewRecorder()
    c.apiService.ServeHTTP(w, req)
    return c.marshalResponse(w, req)
}
```

---

## Pattern summary

| File                                        | What to add                                                              |
| ------------------------------------------- | ------------------------------------------------------------------------ |
| `pkg/mcp/tools/<name>.go`                   | Interface + tool function returning `(mcp.Tool, server.ToolHandlerFunc)` |
| `pkg/mcp/tools/descriptions.go`             | Description constant                                                     |
| `pkg/mcp/client.go`                         | Interface embedded in `Client`                                           |
| `pkg/mcp/api.go`                            | `APIClient` method (HTTP call to agent API)                              |
| `pkg/mcp/server.go`                         | `mcpServer.AddTool(tools.YourTool(client))`                              |
| `cloud-api: internal/server/mcp_handler.go` | `mcpServer.AddTool(mcptools.YourTool(client))`                           |
| `cloud-api: internal/server/mcp_client.go`  | `HandlerClient` method (in-process HTTP via httptest)                    |
