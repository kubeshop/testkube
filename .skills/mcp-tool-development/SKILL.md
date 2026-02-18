---
name: mcp-tool-development
description: Add new tools to the Testkube MCP server. Use when creating new MCP capabilities for AI assistants to interact with Testkube. Covers the interface-based tool design, dual registration requirement (testkube + cloud-api repos), client implementation, and testing with MCP inspector.
metadata:
  author: testkube
  version: "1.0"
---

# MCP Tool Development

MCP server at `pkg/mcp/`. Built on [mcp-go](https://github.com/mark3labs/mcp-go). Tools run via CLI (`testkube mcp serve`) or embedded in the control plane (`/mcp` HTTP endpoint).

**Critical**: Every new tool must be registered in **both** repos. Missing either registration is the #1 mistake.

---

## Step 1: Define the tool interface and handler (testkube repo)

Create `pkg/mcp/tools/my_tool.go`:

```go
package tools

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// Small focused interface — only what this tool needs
type MyEntityGetter interface {
    GetMyEntity(ctx context.Context, id string) (string, error)
}

func GetMyEntity(client MyEntityGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
    tool = mcp.NewTool("get_my_entity",
        mcp.WithDescription("Description of what the tool does and when to use it"),
        mcp.WithString("id", mcp.Required(), mcp.Description("The entity ID")),
    )

    handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        id := request.GetArguments()["id"].(string)

        result, err := client.GetMyEntity(ctx, id)
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        return mcp.NewToolResultText(result), nil
    }

    return tool, handler
}
```

Add the tool description as a constant in `pkg/mcp/tools/descriptions.go`.

## Step 2: Add the interface to Client (testkube repo)

Edit `pkg/mcp/client.go` — add your interface to the composite `Client`:

```go
type Client interface {
    // ... existing interfaces ...
    tools.MyEntityGetter  // Add this
}
```

## Step 3: Implement on APIClient (testkube repo)

Add the method to `pkg/mcp/api.go`:

```go
func (c *APIClient) GetMyEntity(ctx context.Context, id string) (string, error) {
    return c.get(ctx, fmt.Sprintf("/agent/my-entities/%s", id))
}
```

## Step 4: Register the tool in testkube server.go

Edit `pkg/mcp/server.go` — add inside `NewMCPServer()`:

```go
mcpServer.AddTool(tools.GetMyEntity(client))
```

## Step 5: Register the tool in cloud-api mcp_handler.go

Edit `testkube-cloud-api/internal/server/mcp_handler.go` — add inside `createMCPServer()`:

```go
mcpServer.AddTool(mcptools.GetMyEntity(client))
```

## Step 6: Implement on HandlerClient (cloud-api repo)

Edit `testkube-cloud-api/internal/server/mcp_client.go`:

```go
func (c *HandlerClient) GetMyEntity(ctx context.Context, id string) (string, error) {
    req := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/agent/my-entities/%s", id), nil)
    w := httptest.NewRecorder()
    c.apiService.ServeHTTP(w, req)
    return c.marshalResponse(w, req)
}
```

The `HandlerClient` invokes API handlers **in-process** via `httptest.ResponseRecorder` — no network hop.

---

## Build and test

### Build CLI with MCP server

```bash
make build-testkube-cli          # or: make rebuild-kubectl-testkube-cli
```

### Test with MCP Inspector (interactive)

```bash
npx @modelcontextprotocol/inspector ./bin/app/kubectl-testkube mcp serve --debug
```

This opens a web UI where you can invoke tools and see requests/responses.

### Build Docker image

```bash
./build/mcp-server/build-local-mcp.sh testkube/mcp-server:local
```

### Required env vars for Docker

```bash
docker run -e TK_ACCESS_TOKEN=<token> -e TK_ORG_ID=<org> -e TK_ENV_ID=<env> testkube/mcp-server:local
```

---

## Checklist

1. [ ] Tool file in `pkg/mcp/tools/` with small interface + handler function
2. [ ] Description constant in `pkg/mcp/tools/descriptions.go`
3. [ ] Interface added to `Client` in `pkg/mcp/client.go`
4. [ ] `APIClient` method in `pkg/mcp/api.go` (CLI path)
5. [ ] Registration in `pkg/mcp/server.go` `NewMCPServer()`
6. [ ] Registration in `testkube-cloud-api/internal/server/mcp_handler.go` `createMCPServer()`
7. [ ] `HandlerClient` method in `testkube-cloud-api/internal/server/mcp_client.go`
8. [ ] Tested with MCP Inspector

See [references/REFERENCE.md](references/REFERENCE.md) for a complete tool implementation example.
