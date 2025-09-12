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
	Topic       string             `json:"topic"`
	Pages       []DocumentationPage `json:"pages"`
	LearningPath []string          `json:"learning_path"`
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
				URL:         "https://docs.testkube.io/testworkflows/",
				Description: "Introduction to TestWorkflows and their capabilities",
				KeyConcepts: []string{"workflows", "steps", "execution", "orchestration"},
				Level:       "beginner",
			},
			{
				Title:       "Getting Started with TestWorkflows",
				URL:         "https://docs.testkube.io/testworkflows/getting-started",
				Description: "Quick start guide for creating your first TestWorkflow",
				KeyConcepts: []string{"first workflow", "basic structure", "yaml syntax"},
				Level:       "beginner",
			},
		},
		"structure": {
			{
				Title:       "Workflow Structure",
				URL:         "https://docs.testkube.io/testworkflows/workflow-structure",
				Description: "Understanding the basic structure of TestWorkflow YAML files",
				KeyConcepts: []string{"apiVersion", "kind", "metadata", "spec", "steps"},
				Level:       "beginner",
			},
			{
				Title:       "Workflow Metadata",
				URL:         "https://docs.testkube.io/testworkflows/metadata",
				Description: "Configuring workflow metadata, labels, and annotations",
				KeyConcepts: []string{"name", "labels", "annotations", "namespace"},
				Level:       "beginner",
			},
		},
		"steps": {
			{
				Title:       "Workflow Steps",
				URL:         "https://docs.testkube.io/testworkflows/steps",
				Description: "Understanding different types of workflow steps",
				KeyConcepts: []string{"execute", "run", "shell", "conditional", "loops"},
				Level:       "beginner",
			},
			{
				Title:       "Step Execution",
				URL:         "https://docs.testkube.io/testworkflows/step-execution",
				Description: "How steps are executed and their lifecycle",
				KeyConcepts: []string{"execution order", "dependencies", "failure handling"},
				Level:       "intermediate",
			},
		},
		"parallel": {
			{
				Title:       "Parallel Execution",
				URL:         "https://docs.testkube.io/testworkflows/parallel-execution",
				Description: "Running multiple steps or workflows in parallel",
				KeyConcepts: []string{"parallelism", "parallel blocks", "resource management"},
				Level:       "intermediate",
			},
			{
				Title:       "Matrix Execution",
				URL:         "https://docs.testkube.io/testworkflows/matrix-execution",
				Description: "Cross-browser and cross-environment testing patterns",
				KeyConcepts: []string{"matrix", "browser testing", "environment variables"},
				Level:       "advanced",
			},
		},
		"services": {
			{
				Title:       "Services and Dependencies",
				URL:         "https://docs.testkube.io/testworkflows/services",
				Description: "Managing external dependencies like databases and web servers",
				KeyConcepts: []string{"services", "dependencies", "readiness probes", "networking"},
				Level:       "intermediate",
			},
			{
				Title:       "Service Configuration",
				URL:         "https://docs.testkube.io/testworkflows/service-configuration",
				Description: "Advanced service configuration and health checks",
				KeyConcepts: []string{"health checks", "ports", "environment variables", "volumes"},
				Level:       "advanced",
			},
		},
		"artifacts": {
			{
				Title:       "Artifacts and Reports",
				URL:         "https://docs.testkube.io/testworkflows/artifacts",
				Description: "Collecting and managing test artifacts and reports",
				KeyConcepts: []string{"artifacts", "reports", "file collection", "storage"},
				Level:       "intermediate",
			},
			{
				Title:       "JUnit Reports",
				URL:         "https://docs.testkube.io/testworkflows/junit-reports",
				Description: "Generating and collecting JUnit test reports",
				KeyConcepts: []string{"junit", "test reports", "xml", "test results"},
				Level:       "intermediate",
			},
		},
		"configuration": {
			{
				Title:       "Workflow Configuration",
				URL:         "https://docs.testkube.io/testworkflows/configuration",
				Description: "Parameterizing workflows with configuration variables",
				KeyConcepts: []string{"config", "parameters", "variables", "types"},
				Level:       "intermediate",
			},
			{
				Title:       "Environment Variables",
				URL:         "https://docs.testkube.io/testworkflows/environment-variables",
				Description: "Managing environment variables in workflows",
				KeyConcepts: []string{"env", "environment", "secrets", "injection"},
				Level:       "intermediate",
			},
		},
		"templates": {
			{
				Title:       "Workflow Templates",
				URL:         "https://docs.testkube.io/testworkflows/templates",
				Description: "Creating reusable workflow templates",
				KeyConcepts: []string{"templates", "reusability", "parameters", "inheritance"},
				Level:       "advanced",
			},
			{
				Title:       "Template Usage",
				URL:         "https://docs.testkube.io/testworkflows/template-usage",
				Description: "Using and customizing workflow templates",
				KeyConcepts: []string{"template instantiation", "parameter passing", "customization"},
				Level:       "advanced",
			},
		},
		"events": {
			{
				Title:       "Workflow Events",
				URL:         "https://docs.testkube.io/testworkflows/events",
				Description: "Triggering workflows with events and schedules",
				KeyConcepts: []string{"events", "cron", "webhooks", "triggers"},
				Level:       "intermediate",
			},
			{
				Title:       "Cron Jobs",
				URL:         "https://docs.testkube.io/testworkflows/cron-jobs",
				Description: "Scheduling workflows with cron expressions",
				KeyConcepts: []string{"cron", "scheduling", "periodic", "timing"},
				Level:       "intermediate",
			},
		},
		"containers": {
			{
				Title:       "Container Configuration",
				URL:         "https://docs.testkube.io/testworkflows/container-configuration",
				Description: "Configuring containers for workflow execution",
				KeyConcepts: []string{"containers", "images", "resources", "working directory"},
				Level:       "intermediate",
			},
			{
				Title:       "Resource Management",
				URL:         "https://docs.testkube.io/testworkflows/resource-management",
				Description: "Managing CPU and memory resources for workflows",
				KeyConcepts: []string{"resources", "cpu", "memory", "limits", "requests"},
				Level:       "intermediate",
			},
		},
		"content": {
			{
				Title:       "Content Management",
				URL:         "https://docs.testkube.io/testworkflows/content-management",
				Description: "Managing test code and files in workflows",
				KeyConcepts: []string{"content", "git", "files", "repositories"},
				Level:       "beginner",
			},
			{
				Title:       "Git Integration",
				URL:         "https://docs.testkube.io/testworkflows/git-integration",
				Description: "Using Git repositories as content sources",
				KeyConcepts: []string{"git", "repositories", "branches", "commits"},
				Level:       "intermediate",
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
