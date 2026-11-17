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

The MCP server uses your Testkube API key or OAuth login. Standalone HTTP also
requires a separate TK_MCP_SHTTP_AUTH_TOKEN for inbound requests.

Documentation: https://docs.testkube.io/articles/mcp-overview
Configuration: https://docs.testkube.io/articles/mcp-configuration`,
	}

	cmd.AddCommand(NewMcpServeCmd())

	return cmd
}

func NewMcpServeCmd() *cobra.Command {
	var mcpBaseURL string
	var debug bool
	var transport string
	var shttpHost string
	var shttpPort int
	var shttpTLS bool
	var shttpCertFile string
	var shttpKeyFile string

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
• OAuth authentication (run 'testkube login') or a Testkube API key
• For --transport=shttp, a separate TK_MCP_SHTTP_AUTH_TOKEN; clients send it as Authorization: Bearer <token>
• Testkube environment with proper context

The server runs silently by default to avoid interfering with JSON-RPC communication
over stdio. Use --verbose to see detailed output during startup.

Setup Guide: https://docs.testkube.io/articles/mcp-setup
Configuration Examples: https://docs.testkube.io/articles/mcp-configuration`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// Check for environment variable mode (for Docker deployment)
			envMode := os.Getenv("TK_MCP_ENV_MODE") == "true"

			if envMode {
				return runEnvironmentMode(cmd, debug, transport, shttpHost, shttpPort, shttpTLS, shttpCertFile, shttpKeyFile)
			} else {
				return runDefaultMode(cmd, debug, transport, shttpHost, shttpPort, shttpTLS, shttpCertFile, shttpKeyFile)
			}
		},
	}

	cmd.Flags().StringVar(&mcpBaseURL, "base-url", "", "Base URL for Testkube API (uses context API URL if not specified)")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode with detailed operation information")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport protocol: stdio or shttp")
	cmd.Flags().StringVar(&shttpHost, "shttp-host", "localhost", "Host to bind SHTTP server to (external binding must be explicit, including Docker)")
	cmd.Flags().IntVar(&shttpPort, "shttp-port", 8080, "Port to bind SHTTP server to")
	cmd.Flags().BoolVar(&shttpTLS, "shttp-tls", false, "Enable TLS for SHTTP server")
	cmd.Flags().StringVar(&shttpCertFile, "shttp-cert-file", "", "TLS certificate file for SHTTP server")
	cmd.Flags().StringVar(&shttpKeyFile, "shttp-key-file", "", "TLS private key file for SHTTP server")

	return cmd
}

// parseEnvironmentVariables validates and parses required environment variables
func parseEnvironmentVariables() (accessToken, orgID, envID, baseURL, dashboardURL string, err error) {
	accessToken = os.Getenv("TK_ACCESS_TOKEN")
	orgID = os.Getenv("TK_ORG_ID")
	envID = os.Getenv("TK_ENV_ID")
	baseURL = os.Getenv("TK_CONTROL_PLANE_URL")
	dashboardURL = os.Getenv("TK_DASHBOARD_URL")

	if accessToken == "" || orgID == "" || envID == "" {
		return "", "", "", "", "", fmt.Errorf("environment variable mode requires TK_ACCESS_TOKEN, TK_ORG_ID, and TK_ENV_ID")
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

	return accessToken, orgID, envID, baseURL, dashboardURL, nil
}

// displayConfiguration handles verbose output formatting for both modes
func displayConfiguration(config map[string]string, mode string) {
	if !ui.IsVerbose() {
		return
	}

	fmt.Fprintf(os.Stderr, "Starting MCP server in %s mode:\n", mode)
	configJSON, _ := json.MarshalIndent(config, "", "  ")
	fmt.Fprintf(os.Stderr, "Configuration: %s\n", string(configJSON))

	if mode == "default" {
		fmt.Fprintf(os.Stderr, "Configure AI tools: https://docs.testkube.io/articles/mcp-configuration\n")
		fmt.Fprintf(os.Stderr, "Feedback welcome: https://bit.ly/testkube-slack\n")
	}
}

// validateCloudContext validates cloud context, organization ID, and environment ID
func validateCloudContext(cfg config.Data) error {
	if cfg.ContextType != config.ContextTypeCloud {
		return fmt.Errorf("MCP server requires cloud context. Current context: %s. Please run 'testkube set context --help' to configure cloud context. Setup guide: https://docs.testkube.io/articles/mcp-setup", cfg.ContextType)
	}

	if cfg.CloudContext.OrganizationId == "" {
		return fmt.Errorf("organization ID not found in configuration. Please run 'testkube set context' to configure organization")
	}

	if cfg.CloudContext.EnvironmentId == "" {
		return fmt.Errorf("environment ID not found in configuration. Please run 'testkube set context' to configure environment")
	}

	return nil
}

// runEnvironmentMode handles the environment variable mode logic
func runEnvironmentMode(cmd *cobra.Command, debug bool, transport, shttpHost string, shttpPort int, shttpTLS bool, shttpCertFile, shttpKeyFile string) error {
	// Parse environment variables
	accessToken, orgID, envID, baseURL, dashboardURL, err := parseEnvironmentVariables()
	if err != nil {
		return err
	}

	// Prepare configuration display
	configData := map[string]string{
		"Organization":  orgID,
		"Environment":   envID,
		"API Key":       text.Obfuscate(accessToken),
		"API URL":       baseURL,
		"Dashboard URL": dashboardURL,
		"Transport":     transport,
	}

	// Add SHTTP-specific configuration if using SHTTP transport
	if transport == "shttp" {
		configData["SHTTP Host"] = shttpHost
		configData["SHTTP Port"] = fmt.Sprintf("%d", shttpPort)
		configData["SHTTP TLS"] = fmt.Sprintf("%t", shttpTLS)
		if shttpCertFile != "" {
			configData["SHTTP Cert File"] = shttpCertFile
		}
		if shttpKeyFile != "" {
			configData["SHTTP Key File"] = shttpKeyFile
		}
	}

	displayConfiguration(configData, "environment variable")

	// Start the MCP server
	return startMCPServer(common.ResolveSkipTLS(cmd, nil), accessToken, orgID, envID, baseURL, dashboardURL, debug, transport, shttpHost, shttpPort, shttpTLS, shttpCertFile, shttpKeyFile, "cli-env")
}

// runDefaultMode handles the default mode logic (OAuth + config file)
func runDefaultMode(cmd *cobra.Command, debug bool, transport, shttpHost string, shttpPort int, shttpTLS bool, shttpCertFile, shttpKeyFile string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if cfg.CloudContext.ApiKey == "" && !common.IsOAuthAuthenticated() {
		return fmt.Errorf("API key or OAuth authentication required: run 'testkube login' or 'testkube set context --api-key <key>'")
	}
	if err := validateCloudContext(cfg); err != nil {
		return err
	}

	// Get access token
	accessToken, err := common.GetOAuthAccessToken()
	if err != nil {
		accessToken = cfg.CloudContext.ApiKey
		if accessToken == "" {
			return fmt.Errorf("failed to get access token: %w", err)
		}
		if ui.IsVerbose() {
			fmt.Fprintf(os.Stderr, "Using API key for authentication\n")
		}
	} else {
		if ui.IsVerbose() {
			fmt.Fprintf(os.Stderr, "Using OAuth authentication\n")
		}
	}

	// Prepare configuration display
	configData := map[string]string{
		"Organization":  fmt.Sprintf("%s (%s)", cfg.CloudContext.OrganizationName, cfg.CloudContext.OrganizationId),
		"Environment":   fmt.Sprintf("%s (%s)", cfg.CloudContext.EnvironmentName, cfg.CloudContext.EnvironmentId),
		"API URL":       cfg.CloudContext.ApiUri,
		"Dashboard URL": cfg.CloudContext.UiUri,
		"API Key":       text.Obfuscate(accessToken),
	}

	displayConfiguration(configData, "default")

	// Start the MCP server
	return startMCPServer(common.ResolveSkipTLS(cmd, &cfg), accessToken, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId, cfg.CloudContext.ApiUri, cfg.CloudContext.UiUri, debug, transport, shttpHost, shttpPort, shttpTLS, shttpCertFile, shttpKeyFile, "cli-direct")
}

func startMCPServer(skipTLS bool, accessToken, orgID, envID, baseURL, dashboardURL string, debug bool, transport, shttpHost string, shttpPort int, shttpTLS bool, shttpCertFile, shttpKeyFile string, source string) error {
	// Parse transport type
	var transportType mcp.TransportType
	switch transport {
	case "shttp":
		transportType = mcp.TransportSHTTP
	default:
		transportType = mcp.TransportStdio
	}

	// Create MCP server configuration
	mcpCfg := mcp.MCPServerConfig{
		Version:         common.Version,
		Transport:       transportType,
		ControlPlaneUrl: baseURL,
		DashboardUrl:    dashboardURL,
		AccessToken:     accessToken,
		OrgId:           orgID,
		EnvId:           envID,
		Debug:           debug,
		Source:          source,
		SkipTLS:         skipTLS,
		SHTTPConfig: mcp.SHTTPConfig{
			AuthToken: os.Getenv("TK_MCP_SHTTP_AUTH_TOKEN"),
			Host:      shttpHost,
			Port:      shttpPort,
			EnableTLS: shttpTLS,
			CertFile:  shttpCertFile,
			KeyFile:   shttpKeyFile,
		},
	}

	// Fail closed before loading optional settings or creating any API client.
	if err := mcpCfg.Validate(); err != nil {
		return err
	}
	mcpCfg.TelemetryEnabled = true
	if cfg, err := config.Load(); err == nil {
		mcpCfg.TelemetryEnabled = cfg.TelemetryEnabled
	}

	// If no base URL is provided, use the default from testkube context
	if mcpCfg.ControlPlaneUrl == "" {
		mcpCfg.ControlPlaneUrl = "https://api.testkube.io"
	}

	// Start the MCP server using the configured transport
	// The MCP server library handles its own signal management
	if err := mcp.ServeMCP(mcpCfg, nil); err != nil {
		return fmt.Errorf("MCP server error: %v", err)
	}

	return nil
}
