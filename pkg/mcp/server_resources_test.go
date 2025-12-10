package mcp

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/mcp/resources"
)

func TestMCPServerWithResources(t *testing.T) {
	// Create a test configuration
	cfg := MCPServerConfig{
		Version:         "test",
		DashboardUrl:    "https://example.com",
		ControlPlaneUrl: "https://api.example.com",
		OrgId:           "test-org",
		EnvId:           "test-env",
		AccessToken:     "test-token",
		Transport:       TransportStdio,
	}

	// Create MCP server with nil client (will use default)
	server, err := NewMCPServer(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	// Verify resources are registered
	// We can't directly inspect the server's resources, but we can verify
	// that the resource creation functions work
	exampleResources := resources.CreateTestWorkflowExampleResources()
	if len(exampleResources) == 0 {
		t.Fatal("Expected at least one resource to be created")
	}

	expectedCount := 7
	if len(exampleResources) != expectedCount {
		t.Errorf("Expected %d resources, got %d", expectedCount, len(exampleResources))
	}
}
