package mcp

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// TransportType defines the transport protocol for the MCP server
type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportSHTTP TransportType = "shttp"
)

// MCPServerConfig holds configuration for the Testkube MCP server
type MCPServerConfig struct {
	// Version of the server
	Version string

	// Transport specifies the transport protocol to use
	Transport TransportType

	// ControlPlaneUrl for Testkube API
	ControlPlaneUrl string

	// DashboardUrl for Testkube dashboard (by default derived from ControlPlaneUrl)
	DashboardUrl string

	// AccessToken for authenticating with Testkube API
	AccessToken string

	// OrgId for Testkube organization
	OrgId string

	// EnvId for Testkube environment
	EnvId string

	// Debug enables debug mode which includes detailed operation information in responses
	Debug bool

	// TelemetryEnabled enables telemetry collection for MCP tool usage
	TelemetryEnabled bool

	// SHTTP-specific configuration
	SHTTPConfig SHTTPConfig
}

// SHTTPConfig holds configuration for Streamable HTTP transport
type SHTTPConfig struct {
	// Host to bind the HTTP server to (default: "localhost")
	Host string

	// Port to bind the HTTP server to (default: 8080)
	Port int

	// EnableTLS enables HTTPS for the SHTTP server
	EnableTLS bool

	// CertFile path to TLS certificate file (required if EnableTLS is true)
	CertFile string

	// KeyFile path to TLS private key file (required if EnableTLS is true)
	KeyFile string
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() MCPServerConfig {
	controlPlaneUrl := getEnvOrDefault("TK_CONTROL_PLANE_URL", "https://api.testkube.io")

	// Allow explicit dashboard URL override, otherwise derive from control plane URL
	dashboardUrl := os.Getenv("TK_DASHBOARD_URL")
	if dashboardUrl == "" {
		dashboardUrl = controlPlaneUrl
		if re := regexp.MustCompile(`^https?://api\.`); re.MatchString(controlPlaneUrl) {
			dashboardUrl = re.ReplaceAllString(controlPlaneUrl, "https://app.")
		}
	}

	// Parse transport type from environment
	transportStr := getEnvOrDefault("TK_MCP_TRANSPORT", "stdio")
	var transport TransportType
	switch transportStr {
	case "shttp":
		transport = TransportSHTTP
	default:
		transport = TransportStdio
	}

	// Parse SHTTP configuration
	shttpConfig := SHTTPConfig{
		Host:      getEnvOrDefault("TK_MCP_SHTTP_HOST", "localhost"),
		Port:      getEnvOrDefaultAsInt("TK_MCP_SHTTP_PORT", 8080),
		EnableTLS: os.Getenv("TK_MCP_SHTTP_TLS") == "true",
		CertFile:  os.Getenv("TK_MCP_SHTTP_CERT_FILE"),
		KeyFile:   os.Getenv("TK_MCP_SHTTP_KEY_FILE"),
	}

	return MCPServerConfig{
		Version:          "1.0.0",
		Transport:        transport,
		ControlPlaneUrl:  controlPlaneUrl,
		DashboardUrl:     dashboardUrl,
		AccessToken:      os.Getenv("TK_ACCESS_TOKEN"),
		OrgId:            os.Getenv("TK_ORG_ID"),
		EnvId:            os.Getenv("TK_ENV_ID"),
		Debug:            os.Getenv("TK_DEBUG") == "true",
		TelemetryEnabled: os.Getenv("TK_TELEMETRY_ENABLED") != "false", // Default to true unless explicitly disabled
		SHTTPConfig:      shttpConfig,
	}
}

// Validate checks if all required configuration is present
func (c *MCPServerConfig) Validate() error {
	if c.AccessToken == "" {
		return fmt.Errorf("TK_ACCESS_TOKEN is required")
	}

	if c.OrgId == "" {
		return fmt.Errorf("TK_ORG_ID is required")
	}
	if c.EnvId == "" {
		return fmt.Errorf("TK_ENV_ID is required")
	}

	// Validate SHTTP configuration if using SHTTP transport
	if c.Transport == TransportSHTTP {
		if err := c.SHTTPConfig.Validate(); err != nil {
			return fmt.Errorf("SHTTP configuration error: %v", err)
		}
	}

	return nil
}

// Validate checks SHTTP-specific configuration
func (c *SHTTPConfig) Validate() error {
	if c.EnableTLS {
		if c.CertFile == "" {
			return fmt.Errorf("TK_MCP_SHTTP_CERT_FILE is required when TLS is enabled")
		}
		if c.KeyFile == "" {
			return fmt.Errorf("TK_MCP_SHTTP_KEY_FILE is required when TLS is enabled")
		}
	}
	return nil
}

// getEnvOrDefault returns the environment variable value or a default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrDefaultAsInt returns the environment variable value as int or a default if not set
func getEnvOrDefaultAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
