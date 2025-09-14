package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DocumentationPage represents a documentation page reference
type DocumentationPage struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	KeyConcepts []string `json:"key_concepts"`
}

// DocumentationResponse represents the response structure
type DocumentationResponse struct {
	Topic string              `json:"topic"`
	Pages []DocumentationPage `json:"pages"`
}

// GetWorkflowDocumentation returns references to official TestWorkflow documentation
func GetWorkflowDocumentation() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_documentation",
		mcp.WithDescription(GetWorkflowDocumentationDescription),
		mcp.WithString("topic", mcp.Description("Specific topic or concept to find documentation for (e.g., 'parallel', 'services', 'artifacts', 'templates')")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topic := strings.ToLower(request.GetString("topic", ""))

		response := DocumentationResponse{
			Topic: topic,
		}

		// Get documentation pages based on topic
		pages := getDocumentationPages(topic)
		response.Pages = pages

		// Format response as JSON
		result := formatDocumentationResponse(response)

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

// getDocumentationPages returns relevant documentation pages based on topic
func getDocumentationPages(topic string) []DocumentationPage {
	allPages := map[string][]DocumentationPage{
		"": { // General/overview
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Introduction to TestWorkflows and their capabilities",
				KeyConcepts: []string{"workflows", "steps", "execution", "orchestration"},
			},
			{
				Title:       "TestWorkflows Basic Examples",
				URL:         "https://docs.testkube.io/articles/test-workflows-examples-basics",
				Description: "Basic examples to help you get started with TestWorkflows",
				KeyConcepts: []string{"first workflow", "basic structure", "yaml syntax", "examples"},
			},
			{
				Title:       "TestWorkflows CLI Commands",
				URL:         "https://docs.testkube.io/articles/test-workflows-creating",
				Description: "Creating and managing TestWorkflows using CLI commands",
				KeyConcepts: []string{"cli", "creating", "commands", "management"},
			},
		},
		"structure": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Complete overview of TestWorkflow structure, components, and basic concepts",
				KeyConcepts: []string{"apiVersion", "kind", "metadata", "spec", "structure", "components"},
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for TestWorkflow Custom Resource Definition with all properties",
				KeyConcepts: []string{"crd", "reference", "properties", "schema", "validation"},
			},
		},
		"steps": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of TestWorkflow steps, execution patterns, and step types",
				KeyConcepts: []string{"steps", "execute", "run", "shell", "execution patterns"},
			},
			{
				Title:       "TestWorkflows Examples - Basics",
				URL:         "https://docs.testkube.io/articles/test-workflows-examples-basics",
				Description: "Practical examples showing different step types and execution patterns",
				KeyConcepts: []string{"step examples", "execution patterns", "practical examples"},
			},
		},
		"parallel": {
			{
				Title:       "TestWorkflows Parallelization",
				URL:         "https://docs.testkube.io/articles/test-workflows-parallel",
				Description: "Running multiple steps or workflows in parallel for improved performance",
				KeyConcepts: []string{"parallelism", "parallel execution", "concurrent", "performance"},
			},
			{
				Title:       "TestWorkflows Sharding & Matrix Params",
				URL:         "https://docs.testkube.io/articles/test-workflows-matrix-and-sharding",
				Description: "Advanced parallel execution with matrix parameters and sharding patterns",
				KeyConcepts: []string{"matrix", "sharding", "parameters", "advanced parallel"},
			},
		},
		"services": {
			{
				Title:       "TestWorkflows Services",
				URL:         "https://docs.testkube.io/articles/test-workflows-services",
				Description: "Managing service dependencies and integration patterns in TestWorkflows",
				KeyConcepts: []string{"services", "dependencies", "integration", "networking", "probes"},
			},
		},
		"artifacts": {
			{
				Title:       "TestWorkflows Artifacts",
				URL:         "https://docs.testkube.io/articles/test-workflows-artifacts",
				Description: "Collecting and managing test artifacts and reports in TestWorkflows",
				KeyConcepts: []string{"artifacts", "reports", "file collection", "output", "storage"},
			},
		},
		"configuration": {
			{
				Title:       "TestWorkflows Parameterization",
				URL:         "https://docs.testkube.io/articles/test-workflows-examples-configuration",
				Description: "Parameterizing workflows with configuration variables and environment settings",
				KeyConcepts: []string{"config", "parameters", "variables", "environment", "parameterization"},
			},
		},
		"templates": {
			{
				Title:       "TestWorkflow Templates",
				URL:         "https://docs.testkube.io/articles/test-workflow-templates",
				Description: "Creating and using reusable workflow templates for common patterns",
				KeyConcepts: []string{"templates", "reusability", "parameters", "inheritance", "patterns"},
			},
		},
		"events": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of events, triggers, and scheduling in TestWorkflows",
				KeyConcepts: []string{"events", "triggers", "scheduling", "cron"},
			},
		},
		"containers": {
			{
				Title:       "TestWorkflows Job & Pod Configuration",
				URL:         "https://docs.testkube.io/articles/test-workflows-job-and-pod",
				Description: "Configuring containers, jobs, and pods for TestWorkflow execution",
				KeyConcepts: []string{"containers", "jobs", "pods", "images", "resources"},
			},
		},
		"content": {
			{
				Title:       "TestWorkflows Content",
				URL:         "https://docs.testkube.io/articles/test-workflows-content",
				Description: "Managing content sources, Git repositories, and files in TestWorkflows",
				KeyConcepts: []string{"content", "git", "files", "repositories", "sources"},
			},
		},
		"expressions": {
			{
				Title:       "TestWorkflows Expressions",
				URL:         "https://docs.testkube.io/articles/test-workflows-expressions",
				Description: "Using expressions and built-in variables in TestWorkflows",
				KeyConcepts: []string{"expressions", "variables", "built-in", "dynamic", "templating"},
			},
		},
		"policies": {
			{
				Title:       "Enforcing Workflow Policies",
				URL:         "https://docs.testkube.io/articles/enforcing-workflow-policies",
				Description: "Standardizing TestWorkflows with policy enforcement",
				KeyConcepts: []string{"policies", "enforcement", "standardization", "governance"},
			},
		},
		"orchestration": {
			{
				Title:       "TestWorkflows Workflow Orchestration",
				URL:         "https://docs.testkube.io/articles/test-workflows-test-suites",
				Description: "Orchestrating multiple workflows and test suites for complex testing scenarios",
				KeyConcepts: []string{"orchestration", "test suites", "workflow coordination", "complex scenarios"},
			},
		},
		"migration": {
			{
				Title:       "Tests and Test Suites Migration",
				URL:         "https://docs.testkube.io/articles/test-workflow-migration",
				Description: "Migrating from legacy tests and test suites to TestWorkflows",
				KeyConcepts: []string{"migration", "legacy tests", "test suites", "upgrade path"},
			},
		},
	}

	// Get pages for the specific topic
	pages, exists := allPages[topic]
	if !exists {
		// If topic not found, return general pages
		pages = allPages[""]
	}

	return pages
}


// formatDocumentationResponse formats the response as a readable text
func formatDocumentationResponse(response DocumentationResponse) string {
	var result strings.Builder

	result.WriteString("# TestWorkflow Documentation Guide\n\n")

	if response.Topic != "" {
		result.WriteString(fmt.Sprintf("## Topic: %s\n\n", strings.Title(response.Topic)))
	}

	if len(response.Pages) > 0 {
		result.WriteString("## Documentation Pages\n\n")
		for i, page := range response.Pages {
			result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, page.Title))
			result.WriteString(fmt.Sprintf("**URL:** %s\n\n", page.URL))
			result.WriteString(fmt.Sprintf("**Description:** %s\n\n", page.Description))
			if len(page.KeyConcepts) > 0 {
				result.WriteString("**Key Concepts:** ")
				result.WriteString(strings.Join(page.KeyConcepts, ", "))
				result.WriteString("\n\n")
			}
			result.WriteString("---\n\n")
		}
	}

	return result.String()
}
