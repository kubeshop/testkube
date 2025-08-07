package mcp

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/tools"
)

// NewMCPServer creates and configures a new Testkube MCP server
func NewMCPServer(cfg MCPServerConfig, client Client) (*server.MCPServer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	mcpServer := server.NewMCPServer(
		"testkube-mcp",
		cfg.Version,
		server.WithToolCapabilities(true),
		server.WithToolHandlerMiddleware(DebugMiddleware(cfg.Debug)),
	)

	// If no client is provided, use the default API client
	if client == nil {
		httpClient := &http.Client{}
		client = NewAPIClient(&cfg, httpClient)
	}

	// Dashboard tools
	mcpServer.AddTool(tools.BuildDashboardUrl(cfg.DashboardUrl, cfg.OrgId, cfg.EnvId))

	// Workflow tools
	mcpServer.AddTool(tools.ListWorkflows(client))
	mcpServer.AddTool(tools.GetWorkflow(client))
	mcpServer.AddTool(tools.GetWorkflowDefinition(client))
	mcpServer.AddTool(tools.CreateWorkflow(client))
	mcpServer.AddTool(tools.RunWorkflow(client))

	// Execution tools
	mcpServer.AddTool(tools.FetchExecutionLogs(client))
	mcpServer.AddTool(tools.ListExecutions(client))
	mcpServer.AddTool(tools.LookupExecutionId(client))
	mcpServer.AddTool(tools.GetExecutionInfo(client))

	// Artifact tools
	mcpServer.AddTool(tools.ListArtifacts(client))
	mcpServer.AddTool(tools.ReadArtifact(client))

	return mcpServer, nil
}

// ServeStdioMCP creates and starts an MCP server with the given configuration,
// serving over stdio. This is a convenience function that wraps the entire server
// lifecycle so callers don't need to depend on mcp-go directly.
func ServeStdioMCP(cfg MCPServerConfig, client Client) error {
	mcpServer, err := NewMCPServer(cfg, client)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start the server in a goroutine
	serverErrCh := make(chan error, 1)
	go func() {
		defer close(serverErrCh)
		if err := server.ServeStdio(mcpServer); err != nil {
			serverErrCh <- fmt.Errorf("failed to serve MCP server: %v", err)
		}
	}()

	// Wait for either server error or interrupt signal
	select {
	case <-sigCh:
		// Signal received, shutdown gracefully without logging to avoid stdio interference
		return nil // Exit gracefully on interrupt
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
		return nil
	}
}
