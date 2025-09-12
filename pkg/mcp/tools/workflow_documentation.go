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
	Level       string   `json:"level"`
}

// DocumentationResponse represents the response structure
type DocumentationResponse struct {
	Topic        string              `json:"topic"`
	Pages        []DocumentationPage `json:"pages"`
	LearningPath []string            `json:"learning_path"`
}

// GetWorkflowDocumentation returns references to official TestWorkflow documentation
func GetWorkflowDocumentation() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_documentation",
		mcp.WithDescription(GetWorkflowDocumentationDescription),
		mcp.WithString("topic", mcp.Description("Specific topic or concept to find documentation for (e.g., 'parallel', 'services', 'artifacts', 'templates')")),
		mcp.WithString("level", mcp.Description("Documentation level: 'beginner', 'intermediate', 'advanced', or 'all' (default: 'all')")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topic := strings.ToLower(request.GetString("topic", ""))
		level := strings.ToLower(request.GetString("level", "all"))

		response := DocumentationResponse{
			Topic: topic,
		}

		// Get documentation pages based on topic and level
		pages := getDocumentationPages(topic, level)
		response.Pages = pages

		// Add learning path if no specific topic requested
		if topic == "" {
			response.LearningPath = getLearningPath(level)
		}

		// Format response as JSON
		result := formatDocumentationResponse(response)

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

// getDocumentationPages returns relevant documentation pages based on topic and level
func getDocumentationPages(topic, level string) []DocumentationPage {
	allPages := map[string][]DocumentationPage{
		"": { // General/overview
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Introduction to TestWorkflows and their capabilities",
				KeyConcepts: []string{"workflows", "steps", "execution", "orchestration"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflows Examples - Basics",
				URL:         "https://docs.testkube.io/articles/test-workflows-examples-basics",
				Description: "Basic examples to help you get started with TestWorkflows",
				KeyConcepts: []string{"first workflow", "basic structure", "yaml syntax", "examples"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflows High-Level Architecture",
				URL:         "https://docs.testkube.io/articles/test-workflows-high-level-architecture",
				Description: "Understanding the architecture and components of TestWorkflows",
				KeyConcepts: []string{"architecture", "controller", "job", "pod", "components"},
				Level:       "intermediate",
			},
		},
		"structure": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Complete overview of TestWorkflow structure, components, and basic concepts",
				KeyConcepts: []string{"apiVersion", "kind", "metadata", "spec", "structure", "components"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for TestWorkflow Custom Resource Definition with all properties",
				KeyConcepts: []string{"crd", "reference", "properties", "schema", "validation"},
				Level:       "intermediate",
			},
		},
		"steps": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of TestWorkflow steps, execution patterns, and step types",
				KeyConcepts: []string{"steps", "execute", "run", "shell", "execution patterns"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflows Examples - Basics",
				URL:         "https://docs.testkube.io/articles/test-workflows-examples-basics",
				Description: "Practical examples showing different step types and execution patterns",
				KeyConcepts: []string{"step examples", "execution patterns", "practical examples"},
				Level:       "beginner",
			},
		},
		"parallel": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of parallel execution concepts and patterns in TestWorkflows",
				KeyConcepts: []string{"parallelism", "parallel execution", "concurrent", "patterns"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for parallel execution, matrix patterns, and advanced execution",
				KeyConcepts: []string{"parallel", "matrix", "execution", "advanced patterns", "crd"},
				Level:       "intermediate",
			},
		},
		"services": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of service dependencies and integration patterns in TestWorkflows",
				KeyConcepts: []string{"services", "dependencies", "integration", "networking"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for services, dependencies, and networking configuration",
				KeyConcepts: []string{"services", "dependencies", "networking", "probes", "crd"},
				Level:       "intermediate",
			},
		},
		"artifacts": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of artifact collection and reporting in TestWorkflows",
				KeyConcepts: []string{"artifacts", "reports", "file collection", "output"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for artifacts, reports, and file collection configuration",
				KeyConcepts: []string{"artifacts", "reports", "file collection", "storage", "crd"},
				Level:       "intermediate",
			},
		},
		"configuration": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of configuration and environment variables in TestWorkflows",
				KeyConcepts: []string{"config", "parameters", "variables", "environment"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for configuration, parameters, and environment variables",
				KeyConcepts: []string{"config", "parameters", "variables", "environment", "crd"},
				Level:       "intermediate",
			},
		},
		"templates": {
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for workflow templates and reusability patterns",
				KeyConcepts: []string{"templates", "reusability", "parameters", "inheritance", "crd"},
				Level:       "advanced",
			},
			{
				Title:       "TestWorkflows Examples - Basics",
				URL:         "https://docs.testkube.io/articles/test-workflows-examples-basics",
				Description: "Examples showing template usage and reusability patterns",
				KeyConcepts: []string{"template examples", "reusability", "customization"},
				Level:       "intermediate",
			},
		},
		"events": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of events, triggers, and scheduling in TestWorkflows",
				KeyConcepts: []string{"events", "triggers", "scheduling", "cron"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for events, cron jobs, and workflow triggers",
				KeyConcepts: []string{"events", "cron", "webhooks", "triggers", "scheduling", "crd"},
				Level:       "intermediate",
			},
		},
		"containers": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of container configuration and resource management in TestWorkflows",
				KeyConcepts: []string{"containers", "images", "resources", "configuration"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for container configuration and resource management",
				KeyConcepts: []string{"containers", "images", "resources", "cpu", "memory", "crd"},
				Level:       "intermediate",
			},
		},
		"content": {
			{
				Title:       "TestWorkflows Overview",
				URL:         "https://docs.testkube.io/articles/test-workflows",
				Description: "Overview of content management and Git integration in TestWorkflows",
				KeyConcepts: []string{"content", "git", "files", "repositories"},
				Level:       "beginner",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for content management and Git integration",
				KeyConcepts: []string{"content", "git", "files", "repositories", "branches", "crd"},
				Level:       "intermediate",
			},
		},
		"expressions": {
			{
				Title:       "TestWorkflows Expressions",
				URL:         "https://docs.testkube.io/articles/test-workflows-expressions",
				Description: "Using expressions and built-in variables in TestWorkflows",
				KeyConcepts: []string{"expressions", "variables", "built-in", "dynamic", "templating"},
				Level:       "intermediate",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for expressions and variable usage",
				KeyConcepts: []string{"expressions", "variables", "syntax", "reference"},
				Level:       "advanced",
			},
		},
		"policies": {
			{
				Title:       "Enforcing Workflow Policies",
				URL:         "https://docs.testkube.io/articles/enforcing-workflow-policies",
				Description: "Standardizing TestWorkflows with policy enforcement",
				KeyConcepts: []string{"policies", "enforcement", "standardization", "governance"},
				Level:       "advanced",
			},
			{
				Title:       "TestWorkflow CRD Reference",
				URL:         "https://docs.testkube.io/articles/crds/testworkflows.testkube.io-v1",
				Description: "Complete reference for policy configuration and enforcement",
				KeyConcepts: []string{"policies", "configuration", "enforcement", "validation"},
				Level:       "advanced",
			},
		},
	}

	// Get pages for the specific topic
	pages, exists := allPages[topic]
	if !exists {
		// If topic not found, return general pages
		pages = allPages[""]
	}

	// Filter by level if specified
	if level != "all" {
		var filteredPages []DocumentationPage
		for _, page := range pages {
			if page.Level == level {
				filteredPages = append(filteredPages, page)
			}
		}
		pages = filteredPages
	}

	return pages
}

// getLearningPath returns a suggested learning path based on level
func getLearningPath(level string) []string {
	paths := map[string][]string{
		"beginner": {
			"1. Start with TestWorkflows Overview",
			"2. Learn Workflow Structure basics",
			"3. Understand different Step types",
			"4. Explore Content Management",
			"5. Practice with simple workflows",
		},
		"intermediate": {
			"1. Master Parallel Execution",
			"2. Learn about Services and Dependencies",
			"3. Understand Artifacts and Reports",
			"4. Explore Configuration options",
			"5. Set up Events and Triggers",
		},
		"advanced": {
			"1. Master Matrix Execution patterns",
			"2. Create reusable Templates",
			"3. Advanced Resource Management",
			"4. Complex Service Orchestration",
			"5. Custom Container configurations",
		},
		"all": {
			"1. TestWorkflows Overview → Getting Started",
			"2. Workflow Structure → Steps → Content Management",
			"3. Parallel Execution → Services → Artifacts",
			"4. Configuration → Events → Resource Management",
			"5. Matrix Execution → Templates → Advanced Patterns",
		},
	}

	path, exists := paths[level]
	if !exists {
		path = paths["all"]
	}

	return path
}

// formatDocumentationResponse formats the response as a readable text
func formatDocumentationResponse(response DocumentationResponse) string {
	var result strings.Builder

	result.WriteString("# TestWorkflow Documentation Guide\n\n")

	if response.Topic != "" {
		result.WriteString(fmt.Sprintf("## Topic: %s\n\n", strings.Title(response.Topic)))
	}

	if len(response.LearningPath) > 0 {
		result.WriteString("## Learning Path\n\n")
		for _, step := range response.LearningPath {
			result.WriteString(fmt.Sprintf("%s\n", step))
		}
		result.WriteString("\n")
	}

	if len(response.Pages) > 0 {
		result.WriteString("## Documentation Pages\n\n")
		for i, page := range response.Pages {
			result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, page.Title))
			result.WriteString(fmt.Sprintf("**URL:** %s\n\n", page.URL))
			result.WriteString(fmt.Sprintf("**Description:** %s\n\n", page.Description))
			result.WriteString(fmt.Sprintf("**Level:** %s\n\n", strings.Title(page.Level)))
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
