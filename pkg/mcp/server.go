package mcp

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	tkhttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/mcp/tools"
	"github.com/kubeshop/testkube/pkg/ui"
)

const shttpReadHeaderTimeout = 10 * time.Second

// NewMCPServer creates and configures a new Testkube MCP server
func NewMCPServer(cfg MCPServerConfig, client Client) (*server.MCPServer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	mcpServer := server.NewMCPServer(
		"testkube-mcp",
		cfg.Version,
		server.WithRecovery(),
		server.WithToolCapabilities(true),
		server.WithToolHandlerMiddleware(DebugMiddleware(&cfg)),
		server.WithToolHandlerMiddleware(TelemetryMiddleware(&cfg)),
	)

	// If no client is provided, use the default API client
	if client == nil {
		httpClient := tkhttp.NewClient(cfg.SkipTLS)
		client = NewAPIClient(&cfg, httpClient)
	}

	// Dashboard tools
	mcpServer.AddTool(tools.BuildDashboardUrl(cfg.DashboardUrl, cfg.OrgId, cfg.EnvId))

	// Workflow tools
	mcpServer.AddTool(tools.ListWorkflows(client))
	mcpServer.AddTool(tools.GetWorkflow(client))
	mcpServer.AddTool(tools.GetWorkflowDefinition(client))
	mcpServer.AddTool(tools.GetWorkflowMetrics(client))
	mcpServer.AddTool(tools.CreateWorkflow(client))
	mcpServer.AddTool(tools.UpdateWorkflow(client))
	mcpServer.AddTool(tools.RunWorkflow(client))

	// Workflow template tools
	mcpServer.AddTool(tools.ListWorkflowTemplates(client))
	mcpServer.AddTool(tools.GetWorkflowTemplateDefinition(client))
	mcpServer.AddTool(tools.CreateWorkflowTemplate(client))
	mcpServer.AddTool(tools.UpdateWorkflowTemplate(client))

	// Query tools (JSONPath-based bulk queries)
	// Only check backwards compatibility when using APIClient without SkipEndpointChecks
	if apiClient, ok := client.(*APIClient); ok && !cfg.SkipEndpointChecks {
		ctx := context.Background()
		if apiClient.SupportsEndpoint(ctx, "/agent/test-workflows/definitions") {
			mcpServer.AddTool(tools.QueryWorkflows(client))
		}
		if apiClient.SupportsEndpoint(ctx, "/agent/test-workflow-executions/summaries") {
			mcpServer.AddTool(tools.QueryExecutions(client))
		}
	} else {
		mcpServer.AddTool(tools.QueryWorkflows(client))
		mcpServer.AddTool(tools.QueryExecutions(client))
	}

	// Schema tools (static content, no client needed)
	mcpServer.AddTool(tools.GetWorkflowSchema())
	mcpServer.AddTool(tools.GetExecutionSchema())

	// Labels tools
	mcpServer.AddTool(tools.ListLabels(client))

	// Resource groups tools
	mcpServer.AddTool(tools.ListResourceGroups(client))

	// Agent tools
	mcpServer.AddTool(tools.ListAgents(client))

	// Execution tools
	mcpServer.AddTool(tools.FetchExecutionLogs(client))
	mcpServer.AddTool(tools.ListExecutions(client))
	mcpServer.AddTool(tools.LookupExecutionId(client))
	mcpServer.AddTool(tools.GetExecutionInfo(client))
	mcpServer.AddTool(tools.GetWorkflowExecutionMetrics(client))
	mcpServer.AddTool(tools.GetWorkflowResourceHistory(client))
	mcpServer.AddTool(tools.WaitForExecutions(client))
	mcpServer.AddTool(tools.AbortWorkflowExecution(client))
	// Registered unconditionally — endpoint is parameterized and cannot be probed with SupportsEndpoint.
	mcpServer.AddTool(tools.UpdateExecutionTags(client))

	// Artifact tools
	mcpServer.AddTool(tools.ListArtifacts(client))
	mcpServer.AddTool(tools.ReadArtifact(client))

	// Insight (ingested metrics) tools
	mcpServer.AddTool(tools.ListInsightSeries(client))
	mcpServer.AddTool(tools.ListInsightMetricKeys(client))
	mcpServer.AddTool(tools.GetInsightMetricSeries(client))
	mcpServer.AddTool(tools.ListInsightExecutions(client))

	// Insights board tools. Registered unconditionally, like the insight tools:
	// the board routes do not answer HEAD, so SupportsEndpoint cannot probe them.
	mcpServer.AddTool(tools.ListBoards(client))
	mcpServer.AddTool(tools.GetBoard(client))
	mcpServer.AddTool(tools.CreateBoard(client))
	mcpServer.AddTool(tools.UpdateBoard(client))
	mcpServer.AddTool(tools.AddBoardReport(client))
	mcpServer.AddTool(tools.UpdateBoardReport(client))
	mcpServer.AddTool(tools.RemoveBoardReport(client))
	mcpServer.AddTool(tools.DeleteBoard(client))
	mcpServer.AddTool(tools.RenderBoard(client))

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

	if ui.IsVerbose() {
		fmt.Fprintf(os.Stderr, "Testkube MCP server started in stdio mode\n")
		fmt.Fprintf(os.Stderr, "Use 'testkube mcp serve --help' for more options\n")
	}

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

// newSHTTPServer constructs the same authenticated transport used by the CLI and tests.
func newSHTTPServer(cfg MCPServerConfig, client Client) (*server.StreamableHTTPServer, *http.Server, error) {
	// Direct callers must validate HTTP configuration even if Transport was omitted.
	cfg.Transport = TransportSHTTP
	if cfg.SHTTPConfig.Host == "" {
		cfg.SHTTPConfig.Host = "localhost"
	}
	mcpServer, err := NewMCPServer(cfg, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	httpServer := &http.Server{
		Addr:              net.JoinHostPort(cfg.SHTTPConfig.Host, strconv.Itoa(cfg.SHTTPConfig.Port)),
		ReadHeaderTimeout: shttpReadHeaderTimeout,
	}
	opts := []server.StreamableHTTPOption{server.WithStreamableHTTPServer(httpServer)}
	if cfg.SHTTPConfig.EnableTLS {
		opts = append(opts, server.WithTLSCert(cfg.SHTTPConfig.CertFile, cfg.SHTTPConfig.KeyFile))
	}
	transport := server.NewStreamableHTTPServer(mcpServer, opts...)
	mux := http.NewServeMux()
	mux.Handle("/mcp", transport)
	httpServer.Handler = authenticateSHTTP(cfg.SHTTPConfig.AuthToken, mux)
	return transport, httpServer, nil
}

// authenticateSHTTP runs before routing or reading the MCP request body.
func authenticateSHTTP(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.Header.Values("Authorization")
		if len(values) == 1 {
			scheme, credential, ok := strings.Cut(values[0], " ")
			if ok && strings.EqualFold(scheme, "Bearer") && token != "" &&
				subtle.ConstantTimeCompare([]byte(credential), []byte(token)) == 1 {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// ServeSHTTPMCP serves authenticated Streamable HTTP until interrupted.
func ServeSHTTPMCP(cfg MCPServerConfig, client Client) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serveSHTTP(ctx, cfg, client)
}

func serveSHTTP(ctx context.Context, cfg MCPServerConfig, client Client) error {
	transport, httpServer, err := newSHTTPServer(cfg, client)
	if err != nil {
		return err
	}
	serverErrCh := make(chan error, 1)
	go func() { serverErrCh <- transport.Start(httpServer.Addr) }()

	if ui.IsVerbose() {
		protocol := "http"
		if cfg.SHTTPConfig.EnableTLS {
			protocol = "https"
		}
		fmt.Fprintf(os.Stderr, "Testkube MCP endpoint starting on %s://%s/mcp\n", protocol, httpServer.Addr)
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := transport.Shutdown(shutdownCtx); err != nil {
			_ = httpServer.Close()
			return fmt.Errorf("failed to shut down MCP HTTP server: %w", err)
		}
		err = <-serverErrCh
	case err = <-serverErrCh:
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to serve MCP HTTP server: %w", err)
	}
	return nil
}

// ServeMCP starts the MCP server using the configured transport type
func ServeMCP(cfg MCPServerConfig, client Client) error {
	switch cfg.Transport {
	case TransportSHTTP:
		return ServeSHTTPMCP(cfg, client)
	case TransportStdio:
		fallthrough
	default:
		return ServeStdioMCP(cfg, client)
	}
}
