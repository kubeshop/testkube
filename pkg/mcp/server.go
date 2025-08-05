package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/tools"
)

// NewMCPServer creates and configures a new Testkube MCP server
func NewMCPServer(cfg MCPServerConfig, customGetClient tools.GetClientFn) (*server.MCPServer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	mcpServer := server.NewMCPServer(
		"testkube-mcp",
		cfg.Version,
		server.WithToolCapabilities(true),
	)

	// Use custom client if provided, otherwise fall back to default API client
	getClient := customGetClient
	if getClient == nil {
		getClient = func(ctx context.Context) (tools.TestkubeClient, error) {
			httpClient := &http.Client{}
			apiClient := NewAPIClient(&cfg, httpClient)
			return apiClient, nil
		}
	}

	// Dashboard tools
	urlTool, urlHandler := tools.BuildDashboardUrl(cfg.DashboardUrl, cfg.OrgId, cfg.EnvId)
	mcpServer.AddTool(urlTool, urlHandler)

	// Workflow tools
	listWorkflowsTool, listWorkflowsHandler := tools.ListWorkflows(getClient)
	mcpServer.AddTool(listWorkflowsTool, listWorkflowsHandler)

	getWorkflowTool, getWorkflowHandler := tools.GetWorkflow(getClient)
	mcpServer.AddTool(getWorkflowTool, getWorkflowHandler)

	getWorkflowDefinitionTool, getWorkflowDefinitionHandler := tools.GetWorkflowDefinition(getClient)
	mcpServer.AddTool(getWorkflowDefinitionTool, getWorkflowDefinitionHandler)

	createWorkflowTool, createWorkflowHandler := tools.CreateWorkflow(getClient)
	mcpServer.AddTool(createWorkflowTool, createWorkflowHandler)

	runWorkflowTool, runWorkflowHandler := tools.RunWorkflow(getClient)
	mcpServer.AddTool(runWorkflowTool, runWorkflowHandler)

	// Execution tools
	fetchExecutionLogsTool, fetchExecutionLogsHandler := tools.FetchExecutionLogs(getClient)
	mcpServer.AddTool(fetchExecutionLogsTool, fetchExecutionLogsHandler)

	listExecutionsTool, listExecutionsHandler := tools.ListExecutions(getClient)
	mcpServer.AddTool(listExecutionsTool, listExecutionsHandler)

	lookupExecutionIdTool, lookupExecutionIdHandler := tools.LookupExecutionId(getClient)
	mcpServer.AddTool(lookupExecutionIdTool, lookupExecutionIdHandler)

	getExecutionInfoTool, getExecutionInfoHandler := tools.GetExecutionInfo(getClient)
	mcpServer.AddTool(getExecutionInfoTool, getExecutionInfoHandler)

	// Artifact tools
	listArtifactsTool, listArtifactsHandler := tools.ListArtifacts(getClient)
	mcpServer.AddTool(listArtifactsTool, listArtifactsHandler)

	readArtifactTool, readArtifactHandler := tools.ReadArtifact(getClient)
	mcpServer.AddTool(readArtifactTool, readArtifactHandler)

	return mcpServer, nil
}

// ServeStdioMCP creates and starts an MCP server with the given configuration,
// serving over stdio. This is a convenience function that wraps the entire server
// lifecycle so callers don't need to depend on mcp-go directly.
func ServeStdioMCP(cfg MCPServerConfig, customGetClient tools.GetClientFn) error {
	mcpServer, err := NewMCPServer(cfg, customGetClient)
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
