package resources

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestGetTestWorkflowExamples(t *testing.T) {
	examples := GetTestWorkflowExamples()

	if len(examples) == 0 {
		t.Fatal("Expected at least one example, got none")
	}

	expectedCount := 7
	if len(examples) != expectedCount {
		t.Errorf("Expected %d examples, got %d", expectedCount, len(examples))
	}

	// Check that each example has required fields
	for _, example := range examples {
		if example.URI == "" {
			t.Errorf("Example has empty URI")
		}
		if example.Name == "" {
			t.Errorf("Example has empty Name")
		}
		if example.Description == "" {
			t.Errorf("Example has empty Description")
		}
		if example.Content == "" {
			t.Errorf("Example %s has empty Content", example.Name)
		}

		// Check URI format
		if len(example.URI) < len("testworkflow://examples/") {
			t.Errorf("Example %s has invalid URI format: %s", example.Name, example.URI)
		}
	}
}

func TestCreateTestWorkflowExampleResources(t *testing.T) {
	resources := CreateTestWorkflowExampleResources()

	if len(resources) == 0 {
		t.Fatal("Expected at least one resource, got none")
	}

	expectedCount := 7
	if len(resources) != expectedCount {
		t.Errorf("Expected %d resources, got %d", expectedCount, len(resources))
	}

	// Check that each resource has required fields
	for _, resource := range resources {
		if resource.URI == "" {
			t.Errorf("Resource has empty URI")
		}
		if resource.Name == "" {
			t.Errorf("Resource has empty Name")
		}
		if resource.Description == "" {
			t.Errorf("Resource has empty Description")
		}
		if resource.MIMEType != "application/x-yaml" {
			t.Errorf("Resource %s has incorrect MIME type: %s", resource.Name, resource.MIMEType)
		}
	}
}

func TestTestWorkflowExampleResourceHandler(t *testing.T) {
	handler := TestWorkflowExampleResourceHandler()
	examples := GetTestWorkflowExamples()

	if len(examples) == 0 {
		t.Fatal("Expected at least one example to test handler")
	}

	ctx := context.Background()

	// Test reading an existing resource
	firstExample := examples[0]
	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: firstExample.URI,
		},
	}

	contents, err := handler(ctx, request)
	if err != nil {
		t.Fatalf("Failed to read resource %s: %v", firstExample.URI, err)
	}

	if len(contents) == 0 {
		t.Fatalf("Expected content for resource %s, got none", firstExample.URI)
	}

	// Verify content is TextResourceContents
	textContent, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("Expected TextResourceContents, got %T", contents[0])
	}

	if textContent.URI != firstExample.URI {
		t.Errorf("Expected URI %s, got %s", firstExample.URI, textContent.URI)
	}

	if textContent.MIMEType != "application/x-yaml" {
		t.Errorf("Expected MIME type application/x-yaml, got %s", textContent.MIMEType)
	}

	if textContent.Text == "" {
		t.Errorf("Expected non-empty text content for %s", firstExample.URI)
	}

	// Test reading a non-existent resource
	invalidRequest := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "testworkflow://examples/nonexistent",
		},
	}

	_, err = handler(ctx, invalidRequest)
	if err == nil {
		t.Error("Expected error for non-existent resource, got nil")
	}
}

func TestExampleYAMLContents(t *testing.T) {
	examples := GetTestWorkflowExamples()

	for _, example := range examples {
		// Check that content looks like valid YAML
		if len(example.Content) < 10 {
			t.Errorf("Example %s has suspiciously short content: %d bytes", example.Name, len(example.Content))
		}

		// Basic YAML structure checks
		hasApiVersion := false
		hasKind := false
		hasMetadata := false

		lines := splitLines(example.Content)
		for _, line := range lines {
			if len(line) >= 11 && line[:11] == "apiVersion:" {
				hasApiVersion = true
			}
			if len(line) >= 5 && line[:5] == "kind:" {
				hasKind = true
			}
			if len(line) >= 9 && line[:9] == "metadata:" {
				hasMetadata = true
			}
		}

		if !hasApiVersion {
			t.Errorf("Example %s is missing 'apiVersion:' field", example.Name)
		}
		if !hasKind {
			t.Errorf("Example %s is missing 'kind:' field", example.Name)
		}
		if !hasMetadata {
			t.Errorf("Example %s is missing 'metadata:' field", example.Name)
		}
	}
}

// Helper function to split content into lines
func splitLines(content string) []string {
	var lines []string
	current := ""
	for _, c := range content {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
