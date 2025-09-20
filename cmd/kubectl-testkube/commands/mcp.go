package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/mcp"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

func NewMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server enables AI assistants to run workflows, analyze results (Preview)",
		Long: `Model Context Protocol (MCP) server that enables AI assistants to interact with Testkube.

Capabilities:
• Execute and monitor test workflows
• Analyze test results, logs, and artifacts  
• Navigate test execution history
• Manage test resources and configurations

We welcome feedback on Slack: https://bit.ly/testkube-slack

The MCP server requires OAuth authentication and uses your current Testkube context.

Documentation: https://docs.testkube.io/articles/mcp-overview
Configuration: https://docs.testkube.io/articles/mcp-configuration`,
	}

	cmd.AddCommand(NewMcpServeCmd())

	return cmd
}

func NewMcpServeCmd() *cobra.Command {
	var mcpBaseURL string
	var debug bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server for AI assistant integration (runs silently, use --verbose for output)",
		Long: `Start a Model Context Protocol (MCP) server that enables AI assistants to interact with Testkube.

The MCP server provides AI assistants with tools to:
• Execute and monitor test workflows
• Analyze test results, logs, and artifacts
• Navigate test execution history  
• Manage test resources and configurations

Please get in touch on Slack for questions and feedback: https://bit.ly/testkube-slack

Requirements:
• OAuth authentication (run 'testkube login')
• Testkube environment with proper context

The server runs silently by default to avoid interfering with JSON-RPC communication
over stdio. Use --verbose to see detailed output during startup.

Setup Guide: https://docs.testkube.io/articles/mcp-setup
Configuration Examples: https://docs.testkube.io/articles/mcp-configuration`,

		Run: func(cmd *cobra.Command, args []string) {

			// Check for environment variable mode (for Docker deployment)
			envMode := os.Getenv("TK_MCP_ENV_MODE") == "true"

			if envMode {
				// Environment variable mode - use env vars directly
				startMCPServerInEnvMode(debug)
				return
			}

			// Load configuration to get org and env IDs
			cfg, err := config.Load()
			if err != nil {
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
				}
				return
			}

			// OAuth authentication check (default mode)
			if cfg.CloudContext.ApiKey == "" && !common.IsOAuthAuthenticated() {
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "API key or OAuth authentication required\n")
					fmt.Fprintf(os.Stderr, "Please run 'testkube login' to authenticate with OAuth flow\n")
					fmt.Fprintf(os.Stderr, "Or set API key using 'testkube set context'\n")
					fmt.Fprintf(os.Stderr, "Setup guide: https://docs.testkube.io/articles/mcp-setup\n")
				}
				return
			}

			// Validate we have required context information
			if cfg.ContextType != config.ContextTypeCloud {
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "MCP server requires cloud context. Current context: %s\n", cfg.ContextType)
					fmt.Fprintf(os.Stderr, "Please run 'testkube set context --help' to configure cloud context\n")
					fmt.Fprintf(os.Stderr, "Setup guide: https://docs.testkube.io/articles/mcp-setup\n")
				}
				return
			}

			if cfg.CloudContext.OrganizationId == "" {
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "Organization ID not found in configuration\n")
					fmt.Fprintf(os.Stderr, "Please run 'testkube set context' to configure organization\n")
				}
				return
			}

			if cfg.CloudContext.EnvironmentId == "" {
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "Environment ID not found in configuration\n")
					fmt.Fprintf(os.Stderr, "Please run 'testkube set context' to configure environment\n")
				}
				return
			}

			// Get the current access token
			accessToken, err := common.GetOAuthAccessToken()
			if err != nil {

				accessToken = cfg.CloudContext.ApiKey
				if accessToken == "" {
					if ui.IsVerbose() {
						fmt.Fprintf(os.Stderr, "Failed to get access token: %v\n", err)
					}
					return
				}
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "Using API key for authentication\n")
				}
			} else {
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "Using OAuth authentication\n")
				}
			}

			// Display connection information
			if ui.IsVerbose() {
				fmt.Fprintf(os.Stderr, "Starting MCP server with configuration:\n")
				configData := map[string]string{
					"Organization":  fmt.Sprintf("%s (%s)", cfg.CloudContext.OrganizationName, cfg.CloudContext.OrganizationId),
					"Environment":   fmt.Sprintf("%s (%s)", cfg.CloudContext.EnvironmentName, cfg.CloudContext.EnvironmentId),
					"API URL":       cfg.CloudContext.ApiUri,
					"Dashboard URL": cfg.CloudContext.UiUri,
					"API Key":       text.Obfuscate(accessToken),
				}
				configJSON, _ := json.MarshalIndent(configData, "", "  ")
				fmt.Fprintf(os.Stderr, "Configuration: %s\n", string(configJSON))
				fmt.Fprintf(os.Stderr, "Configure AI tools: https://docs.testkube.io/articles/mcp-configuration\n")
				fmt.Fprintf(os.Stderr, "Feedback welcome: https://bit.ly/testkube-slack\n")
			}

			// Start the MCP server
			if err := startMCPServer(accessToken, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId, cfg.CloudContext.ApiUri, cfg.CloudContext.UiUri, debug); err != nil {
				if ui.IsVerbose() {
					fmt.Fprintf(os.Stderr, "Failed to start MCP server: %v\n", err)
				}
				return
			}

			// If we reach here, the server shut down gracefully
			if ui.IsVerbose() {
				fmt.Fprintf(os.Stderr, "MCP server shut down gracefully\n")
			}
		},
	}

	cmd.Flags().StringVar(&mcpBaseURL, "base-url", "", "Base URL for Testkube API (uses context API URL if not specified)")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode with detailed operation information")

	return cmd
}

func startMCPServerInEnvMode(debug bool) {
	accessToken := os.Getenv("TK_ACCESS_TOKEN")
	orgID := os.Getenv("TK_ORG_ID")
	envID := os.Getenv("TK_ENV_ID")
	baseURL := os.Getenv("TK_CONTROL_PLANE_URL")
	dashboardURL := os.Getenv("TK_DASHBOARD_URL")

	if accessToken == "" || orgID == "" || envID == "" {
		if ui.IsVerbose() {
			fmt.Fprintf(os.Stderr, "Environment variable mode requires TK_ACCESS_TOKEN, TK_ORG_ID, and TK_ENV_ID\n")
			fmt.Fprintf(os.Stderr, "Set TK_MCP_ENV_MODE=true to enable environment variable mode\n")
		}
	}

	if baseURL == "" {
		baseURL = "https://api.testkube.io"
	}
	if dashboardURL == "" {
		dashboardURL = baseURL
		if strings.HasPrefix(baseURL, "https://api.") {
			dashboardURL = strings.Replace(baseURL, "https://api.", "https://app.", 1)
		}
	}

	if ui.IsVerbose() {
		fmt.Fprintf(os.Stderr, "Starting MCP server in environment variable mode:\n")
		envConfigData := map[string]string{
			"Organization":  orgID,
			"Environment":   envID,
			"API Key":       text.Obfuscate(accessToken),
			"API URL":       baseURL,
			"Dashboard URL": dashboardURL,
		}
		envConfigJSON, _ := json.MarshalIndent(envConfigData, "", "  ")
		fmt.Fprintf(os.Stderr, "Configuration: %s\n", string(envConfigJSON))
	}

	// Start the MCP server with environment variables
	if err := startMCPServer(accessToken, orgID, envID, baseURL, dashboardURL, debug); err != nil {
		if ui.IsVerbose() {
			fmt.Fprintf(os.Stderr, "Failed to start MCP server: %v\n", err)
		}
	}
}

func startMCPServer(accessToken, orgID, envID, baseURL, dashboardURL string, debug bool) error {
	// Create MCP server configuration
	mcpCfg := mcp.MCPServerConfig{
		Version:         common.Version,
		ControlPlaneUrl: baseURL,
		DashboardUrl:    dashboardURL,
		AccessToken:     accessToken,
		OrgId:           orgID,
		EnvId:           envID,
		Debug:           debug,
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
