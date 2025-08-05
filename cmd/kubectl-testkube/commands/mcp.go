package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/mcp"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage Model Context Protocol (MCP) server for Testkube",
		Long: `Manage Model Context Protocol (MCP) server that exposes Testkube functionality.

The MCP server requires OAuth authentication and will use the current Testkube context
to determine the organization and environment to connect to.`,
	}

	cmd.AddCommand(NewMcpServeCmd())

	return cmd
}

func NewMcpServeCmd() *cobra.Command {
	var mcpBaseURL string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server for Testkube (silent by default, use --verbose for output)",
		Long: `Start a Model Context Protocol (MCP) server that exposes Testkube functionality.

The MCP server requires OAuth authentication and will use the current Testkube context
to determine the organization and environment to connect to.

The server runs silently by default to avoid interfering with JSON-RPC communication
over stdio. Use --verbose to see detailed output during startup.`,

		Run: func(cmd *cobra.Command, args []string) {
			// OAuth authentication check
			if !common.IsOAuthAuthenticated() {
				if ui.IsVerbose() {
					ui.Failf("OAuth authentication required")
					ui.Info("Please run 'testkube login' to authenticate with OAuth flow")
				}
				return
			}

			// Load configuration to get org and env IDs
			cfg, err := config.Load()
			if err != nil {
				if ui.IsVerbose() {
					ui.Failf("Failed to load configuration: %v", err)
				}
				return
			}

			// Validate we have required context information
			if cfg.ContextType != config.ContextTypeCloud {
				if ui.IsVerbose() {
					ui.Failf("MCP server requires cloud context. Current context: %s", cfg.ContextType)
					ui.Info("Please run 'testkube set context --help' to configure cloud context")
				}
				return
			}

			if cfg.CloudContext.OrganizationId == "" {
				if ui.IsVerbose() {
					ui.Failf("Organization ID not found in configuration")
					ui.Info("Please run 'testkube set context' to configure organization")
				}
				return
			}

			if cfg.CloudContext.EnvironmentId == "" {
				if ui.IsVerbose() {
					ui.Failf("Environment ID not found in configuration")
					ui.Info("Please run 'testkube set context' to configure environment")
				}
				return
			}

			// Get the current access token
			accessToken, err := common.GetOAuthAccessToken()
			if err != nil {
				if ui.IsVerbose() {
					ui.Failf("Failed to get access token: %v", err)
				}
				return
			}

			// Display connection information
			if ui.IsVerbose() {
				ui.Info("Starting MCP server with configuration:")
				ui.InfoGrid(map[string]string{
					"Organization":  fmt.Sprintf("%s (%s)", cfg.CloudContext.OrganizationName, cfg.CloudContext.OrganizationId),
					"Environment":   fmt.Sprintf("%s (%s)", cfg.CloudContext.EnvironmentName, cfg.CloudContext.EnvironmentId),
					"API URL":       cfg.CloudContext.ApiUri,
					"Dashboard URL": cfg.CloudContext.UiUri,
				})
			}

			// Start the MCP server
			if err := startMCPServer(accessToken, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId, cfg.CloudContext.ApiUri, cfg.CloudContext.UiUri); err != nil {
				if ui.IsVerbose() {
					ui.Failf("Failed to start MCP server: %v", err)
				}
				return
			}

			// If we reach here, the server shut down gracefully
			if ui.IsVerbose() {
				ui.Info("MCP server shut down gracefully")
			}
		},
	}

	cmd.Flags().StringVar(&mcpBaseURL, "base-url", "", "Base URL for Testkube API (uses context API URL if not specified)")

	return cmd
}

func startMCPServer(accessToken, orgID, envID, baseURL, dashboardURL string) error {
	// Create MCP server configuration
	mcpCfg := mcp.MCPServerConfig{
		Version:         "1.0.0",
		ControlPlaneUrl: baseURL,
		DashboardUrl:    dashboardURL,
		AccessToken:     accessToken,
		OrgId:           orgID,
		EnvId:           envID,
	}

	// If no base URL is provided, use the default from testkube context
	if mcpCfg.ControlPlaneUrl == "" {
		mcpCfg.ControlPlaneUrl = "https://api.testkube.io"
	}

	// Start the MCP server - this will block and handle stdio
	// The MCP server library handles its own signal management
	if err := mcp.ServeStdioMCP(mcpCfg, nil); err != nil {
		return fmt.Errorf("MCP server error: %v", err)
	}

	return nil
}
