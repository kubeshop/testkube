package mcp

import (
	"fmt"
	"os"
	"regexp"
)

// MCPServerConfig holds configuration for the Testkube MCP server
type MCPServerConfig struct {
	// Version of the server
	Version string

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

	return MCPServerConfig{
		Version:         "1.0.0",
		ControlPlaneUrl: controlPlaneUrl,
		DashboardUrl:    dashboardUrl,
		AccessToken:     os.Getenv("TK_ACCESS_TOKEN"),
		OrgId:           os.Getenv("TK_ORG_ID"),
		EnvId:           os.Getenv("TK_ENV_ID"),
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
	return nil
}

// getEnvOrDefault returns the environment variable value or a default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
