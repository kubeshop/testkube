package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TestWorkflowExample represents a TestWorkflow example resource
type TestWorkflowExample struct {
	URI         string
	Name        string
	Description string
	FilePath    string // Relative path from repository root
}

// GetTestWorkflowExamples returns a list of TestWorkflow example resources
// These examples point to actual test files from the test/ directory
func GetTestWorkflowExamples() []TestWorkflowExample {
	return []TestWorkflowExample{
		{
			URI:         "testworkflow://examples/postman-smoke",
			Name:        "Postman Smoke Test Workflows",
			Description: "Multiple Postman workflow examples including simple runs, templates, and JUnit reporting. File: test/postman/crd-workflow/smoke.yaml",
			FilePath:    "test/postman/crd-workflow/smoke.yaml",
		},
		{
			URI:         "testworkflow://examples/playwright-smoke",
			Name:        "Playwright E2E Test Workflows",
			Description: "Playwright workflows demonstrating E2E testing with artifacts, JUnit reports, and trace collection. File: test/playwright/crd-workflow/smoke.yaml",
			FilePath:    "test/playwright/crd-workflow/smoke.yaml",
		},
		{
			URI:         "testworkflow://examples/k6-smoke",
			Name:        "K6 Load Testing Workflows",
			Description: "K6 workflows for performance and load testing with various configurations. File: test/k6/crd-workflow/smoke.yaml",
			FilePath:    "test/k6/crd-workflow/smoke.yaml",
		},
		{
			URI:         "testworkflow://examples/special-cases",
			Name:        "Special Cases and Advanced Features",
			Description: "Demonstrates advanced features: ENV overrides, retries, conditions, parallel execution, shared volumes, security contexts. File: test/special-cases/special-cases.yaml",
			FilePath:    "test/special-cases/special-cases.yaml",
		},
	}
}

// getRepoRoot returns the repository root directory
// It walks up from the current file location to find the repo root
func getRepoRoot() (string, error) {
	// Get the directory of this source file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get current file path")
	}
	
	// Start from this file's directory and walk up
	dir := filepath.Dir(filename)
	for {
		// Check if this directory contains go.mod (indicator of repo root)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		
		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod
			return "", fmt.Errorf("could not find repository root (go.mod not found)")
		}
		dir = parent
	}
}

// loadExample loads an example file from the test directory
func loadExample(filePath string) (string, error) {
	// Get repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find repository root: %w", err)
	}
	
	// Construct full path
	fullPath := filepath.Join(repoRoot, filePath)
	
	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}
	
	return string(content), nil
}

// CreateTestWorkflowExampleResources creates MCP resources for TestWorkflow examples
func CreateTestWorkflowExampleResources() []mcp.Resource {
	examples := GetTestWorkflowExamples()
	resources := make([]mcp.Resource, len(examples))

	for i, example := range examples {
		resources[i] = mcp.NewResource(
			example.URI,
			example.Name,
			mcp.WithResourceDescription(example.Description),
			mcp.WithMIMEType("application/x-yaml"),
		)
	}

	return resources
}

// TestWorkflowExampleResourceHandler creates a resource handler for TestWorkflow examples
func TestWorkflowExampleResourceHandler() server.ResourceHandlerFunc {
	examples := GetTestWorkflowExamples()
	
	// Create a map for quick lookup
	exampleMap := make(map[string]TestWorkflowExample)
	for _, example := range examples {
		exampleMap[example.URI] = example
	}

	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		example, exists := exampleMap[request.Params.URI]
		if !exists {
			return nil, fmt.Errorf("resource not found: %s", request.Params.URI)
		}

		// Load content from the test file
		content, err := loadExample(example.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load resource %s: %w", request.Params.URI, err)
		}

		// Return the example content as TextResourceContents
		contents := mcp.TextResourceContents{
			URI:      example.URI,
			MIMEType: "application/x-yaml",
			Text:     content,
		}

		return []mcp.ResourceContents{contents}, nil
	}
}
