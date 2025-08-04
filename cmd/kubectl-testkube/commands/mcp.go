package commands

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"

	mcpconfig "github.com/kubeshop/testkube-mcp/config"
	mcpserver "github.com/kubeshop/testkube-mcp/server"
)

func NewMcpCmd() *cobra.Command {
	var mcpMode string
	var mcpBaseURL string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for Testkube",
		Long: `Start a Model Context Protocol (MCP) server that exposes Testkube functionality.

The MCP server requires OAuth authentication and will use the current Testkube context
to determine the organization and environment to connect to.`,

		Run: func(cmd *cobra.Command, args []string) {
			// OAuth authentication check
			if !common.IsOAuthAuthenticated() {
				ui.Failf("OAuth authentication required")
				ui.Info("Please run 'testkube pro login' to authenticate with OAuth flow")
				return
			}

			ui.Success("OAuth authentication validated")

			// Load configuration to get org and env IDs
			cfg, err := config.Load()
			if err != nil {
				ui.Failf("Failed to load configuration: %v", err)
				return
			}

			// Validate we have required context information
			if cfg.ContextType != config.ContextTypeCloud {
				ui.Failf("MCP server requires cloud context. Current context: %s", cfg.ContextType)
				ui.Info("Please run 'testkube set context --help' to configure cloud context")
				return
			}

			if cfg.CloudContext.OrganizationId == "" {
				ui.Failf("Organization ID not found in configuration")
				ui.Info("Please run 'testkube set context' to configure organization")
				return
			}

			if cfg.CloudContext.EnvironmentId == "" {
				ui.Failf("Environment ID not found in configuration")
				ui.Info("Please run 'testkube set context' to configure environment")
				return
			}

			// Get the current access token
			accessToken, err := common.GetOAuthAccessToken()
			if err != nil {
				ui.Failf("Failed to get access token: %v", err)
				return
			}

			// Display connection information
			ui.Info("Starting MCP server with configuration:")
			ui.InfoGrid(map[string]string{
				"Organization": fmt.Sprintf("%s (%s)", cfg.CloudContext.OrganizationName, cfg.CloudContext.OrganizationId),
				"Environment":  fmt.Sprintf("%s (%s)", cfg.CloudContext.EnvironmentName, cfg.CloudContext.EnvironmentId),
				"API URL":      cfg.CloudContext.ApiUri,
				"Mode":         mcpMode,
			})

			// Start the MCP server
			if err := startMCPServer(accessToken, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId, mcpMode, mcpBaseURL); err != nil {
				ui.Failf("Failed to start MCP server: %v", err)
				return
			}
		},
	}

	cmd.Flags().StringVar(&mcpMode, "mode", "api", "MCP server mode (api or handler)")
	cmd.Flags().StringVar(&mcpBaseURL, "base-url", "", "Base URL for Testkube API (uses context API URL if not specified)")

	return cmd
}

func startMCPServer(accessToken, orgID, envID, mode, baseURL string) error {
	// Create MCP server configuration
	mcpCfg := mcpconfig.MCPServerConfig{
		Version:  "1.0.0",
		Mode:     mode,
		BaseURL:  baseURL,
		APIToken: accessToken,
		OrgID:    orgID,
		EnvID:    envID,
	}

	// If no base URL is provided, use the default from testkube context
	if mcpCfg.BaseURL == "" {
		mcpCfg.BaseURL = "https://api.testkube.io"
	}

	// Write initial info to stderr so it doesn't interfere with MCP stdio protocol
	fmt.Fprintf(os.Stderr, "Starting Testkube MCP server...\n")
	fmt.Fprintf(os.Stderr, "Configuration: Mode=%s, OrgID=%s, EnvID=%s\n", mode, orgID, envID)
	fmt.Fprintf(os.Stderr, "MCP server is now ready for communication.\n")

	// Start the MCP server - this will block and handle stdio
	// The MCP server library handles its own signal management
	if err := mcpserver.StartMCPServerWithStdio(mcpCfg); err != nil {
		return fmt.Errorf("MCP server error: %v", err)
	}

	return nil
}
