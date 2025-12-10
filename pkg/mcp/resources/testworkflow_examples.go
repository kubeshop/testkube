package resources

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed examples/*.yaml
var examplesFS embed.FS

// TestWorkflowExample represents a TestWorkflow example resource
type TestWorkflowExample struct {
	URI         string
	Name        string
	Description string
	Content     string
}

// GetTestWorkflowExamples returns a list of TestWorkflow example resources
func GetTestWorkflowExamples() []TestWorkflowExample {
	return []TestWorkflowExample{
		{
			URI:         "testworkflow://examples/postman-simple",
			Name:        "Postman Simple Workflow",
			Description: "A simple Postman workflow that runs a collection with environment variables",
			Content:     loadExample("postman-simple.yaml"),
		},
		{
			URI:         "testworkflow://examples/playwright-e2e",
			Name:        "Playwright E2E Workflow",
			Description: "A Playwright workflow with artifacts, JUnit reports, and trace collection",
			Content:     loadExample("playwright-e2e.yaml"),
		},
		{
			URI:         "testworkflow://examples/k6-load-test",
			Name:        "K6 Load Testing Workflow",
			Description: "A k6 workflow for load testing with custom configuration",
			Content:     loadExample("k6-load-test.yaml"),
		},
		{
			URI:         "testworkflow://examples/special-cases-env-override",
			Name:        "Special Cases: ENV Variable Override",
			Description: "Demonstrates ENV variable overrides at different levels (container, step, parallel)",
			Content:     loadExample("special-cases-env-override.yaml"),
		},
		{
			URI:         "testworkflow://examples/special-cases-retries",
			Name:        "Special Cases: Retries and Conditions",
			Description: "Demonstrates retry logic and conditional step execution",
			Content:     loadExample("special-cases-retries.yaml"),
		},
		{
			URI:         "testworkflow://examples/parallel-execution",
			Name:        "Parallel Execution Workflow",
			Description: "Demonstrates parallel step execution and matrix workflows",
			Content:     loadExample("parallel-execution.yaml"),
		},
		{
			URI:         "testworkflow://examples/artifacts-and-junit",
			Name:        "Artifacts and JUnit Reports",
			Description: "Demonstrates artifact collection and JUnit report generation",
			Content:     loadExample("artifacts-and-junit.yaml"),
		},
	}
}

// loadExample loads an example file from the embedded filesystem
func loadExample(filename string) string {
	content, err := examplesFS.ReadFile(filepath.Join("examples", filename))
	if err != nil {
		// Return a placeholder if the file doesn't exist yet
		return fmt.Sprintf("# Example file %s not found\n# This is a placeholder that will be replaced with actual content", filename)
	}
	return string(content)
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

		// Return the example content as TextResourceContents
		contents := mcp.TextResourceContents{
			URI:      example.URI,
			MIMEType: "application/x-yaml",
			Text:     example.Content,
		}

		return []mcp.ResourceContents{contents}, nil
	}
}
