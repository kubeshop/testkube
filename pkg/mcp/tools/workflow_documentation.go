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

// WorkflowPattern represents a common workflow pattern with examples
type WorkflowPattern struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	YAMLExample string   `json:"yaml_example"`
	UseCases    []string `json:"use_cases"`
	Complexity  string   `json:"complexity"` // "basic", "intermediate", "advanced"
	Tags        []string `json:"tags"`
}

// StepType represents a step type with its properties and examples
type StepType struct {
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Properties  map[string]Property `json:"properties"`
	Examples    []string            `json:"examples"`
	Required    []string            `json:"required"`
	Optional    []string            `json:"optional"`
	YAMLSnippet string              `json:"yaml_snippet"`
}

// Property represents a step property definition
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
	Example     string `json:"example,omitempty"`
	Required    bool   `json:"required"`
}

// BestPractice represents a best practice or validation rule
type BestPractice struct {
	Category     string   `json:"category"`
	Rule         string   `json:"rule"`
	Description  string   `json:"description"`
	Examples     []string `json:"examples"`
	AntiPatterns []string `json:"anti_patterns"`
}

// EnhancedDocumentationResponse represents the enhanced response structure
type EnhancedDocumentationResponse struct {
	Topic          string              `json:"topic"`
	Pages          []DocumentationPage `json:"pages"`
	Patterns       []WorkflowPattern   `json:"patterns,omitempty"`
	StepTypes      []StepType          `json:"step_types,omitempty"`
	BestPractices  []BestPractice      `json:"best_practices,omitempty"`
	QuickStart     []string            `json:"quick_start,omitempty"`
	CommonMistakes []string            `json:"common_mistakes,omitempty"`
}

// GetWorkflowDocumentation returns references to official TestWorkflow documentation
func GetWorkflowDocumentation() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_documentation",
		mcp.WithDescription(GetWorkflowDocumentationDescription),
		mcp.WithString("topic", mcp.Description("Specific topic or concept to find documentation for. Available topics: structure, steps, parallel, services, artifacts, configuration, templates, events, containers, content, expressions, policies, orchestration, migration. Leave empty for general overview.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topic := strings.ToLower(request.GetString("topic", ""))

		response := EnhancedDocumentationResponse{
			Topic: topic,
		}

		// Get documentation pages based on topic
		pages := getDocumentationPages(topic)
		response.Pages = pages

		// Get workflow patterns based on topic
		patterns := getWorkflowPatterns(topic)
		response.Patterns = patterns

		// Get step types based on topic
		stepTypes := getStepTypes(topic)
		response.StepTypes = stepTypes

		// Get best practices based on topic
		bestPractices := getBestPractices(topic)
		response.BestPractices = bestPractices

		// Get quick start guide
		quickStart := getQuickStartGuide(topic)
		response.QuickStart = quickStart

		// Get common mistakes
		commonMistakes := getCommonMistakes(topic)
		response.CommonMistakes = commonMistakes

		// Format response as JSON
		result := formatEnhancedDocumentationResponse(response)

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

// getWorkflowPatterns returns relevant workflow patterns based on topic
func getWorkflowPatterns(topic string) []WorkflowPattern {
	allPatterns := map[string][]WorkflowPattern{
		"": { // General patterns
			{
				Name:        "Basic Shell Execution",
				Description: "Simple workflow that runs shell commands",
				YAMLExample: `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: basic-shell-test
spec:
  container:
    image: alpine:3.22.0
  steps:
  - name: Run test
    shell: echo "Hello World" && exit 0`,
				UseCases:   []string{"Simple scripts", "Basic validation", "Quick tests"},
				Complexity: "basic",
				Tags:       []string{"shell", "basic", "alpine"},
			},
			{
				Name:        "Parallel Matrix Execution",
				Description: "Run the same test with different parameters in parallel",
				YAMLExample: `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: matrix-test
spec:
  container:
    image: curlimages/curl:8.7.1
  steps:
  - name: Test multiple URLs
    parallel:
      matrix:
        url:
          - "https://example.com"
          - "https://testkube.io"
          - "https://docs.testkube.io"
    shell: curl -f -LI '{{ matrix.url }}'`,
				UseCases:   []string{"Multi-environment testing", "Parameterized tests", "Load distribution"},
				Complexity: "intermediate",
				Tags:       []string{"parallel", "matrix", "curl"},
			},
		},
		"steps": {
			{
				Name:        "Conditional Step Execution",
				Description: "Execute steps based on conditions",
				YAMLExample: `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: conditional-steps
spec:
  container:
    image: alpine:3.22.0
  steps:
  - name: Setup
    shell: echo "Setting up environment"
  - name: Conditional test
    condition: "{{ steps.setup.result == 'passed' }}"
    shell: echo "Running conditional test"
  - name: Always run cleanup
    condition: "always"
    shell: echo "Cleaning up"`,
				UseCases:   []string{"Conditional logic", "Error handling", "Cleanup operations"},
				Complexity: "intermediate",
				Tags:       []string{"conditional", "logic", "cleanup"},
			},
		},
		"parallel": {
			{
				Name:        "Matrix Parameter Testing",
				Description: "Test multiple combinations of parameters",
				YAMLExample: `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: matrix-browser-test
spec:
  container:
    image: cypress/included:14.5.1
  steps:
  - name: Cross-browser testing
    parallel:
      matrix:
        browser: ["chrome", "firefox", "edge"]
        viewport: ["1920x1080", "1366x768", "375x667"]
    shell: |
      echo "Testing {{ matrix.browser }} at {{ matrix.viewport }}"
      # Add actual test commands here`,
				UseCases:   []string{"Cross-browser testing", "Multi-configuration testing", "Compatibility testing"},
				Complexity: "advanced",
				Tags:       []string{"matrix", "browser", "viewport", "cypress"},
			},
		},
		"services": {
			{
				Name:        "Database Integration Test",
				Description: "Test with database service dependency",
				YAMLExample: `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: database-integration-test
spec:
  container:
    image: postgres:15-alpine
  services:
    postgres:
      image: postgres:15-alpine
      env:
        POSTGRES_DB: testdb
        POSTGRES_USER: testuser
        POSTGRES_PASSWORD: testpass
      readinessProbe:
        exec:
          command: ["pg_isready", "-U", "testuser", "-d", "testdb"]
  steps:
  - name: Wait for database
    shell: |
      until pg_isready -h postgres -U testuser -d testdb; do
        echo "Waiting for database..."
        sleep 2
      done
  - name: Run database tests
    shell: |
      psql -h postgres -U testuser -d testdb -c "SELECT 1;"`,
				UseCases:   []string{"Database testing", "Integration testing", "Service dependencies"},
				Complexity: "intermediate",
				Tags:       []string{"database", "postgres", "integration", "services"},
			},
		},
		"artifacts": {
			{
				Name:        "Test Report Collection",
				Description: "Collect and store test artifacts and reports",
				YAMLExample: `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: report-collection-test
spec:
  container:
    image: cypress/included:14.5.1
  steps:
  - name: Run tests
    shell: npx cypress run --reporter junit --reporter-options "mochaFile=results/results.xml"
  - name: Collect artifacts
    artifacts:
      paths:
        - "results/**/*"
        - "cypress/screenshots/**/*"
        - "cypress/videos/**/*"
    workingDir: /data/artifacts`,
				UseCases:   []string{"Test reporting", "Screenshot collection", "Video recording", "JUnit reports"},
				Complexity: "intermediate",
				Tags:       []string{"artifacts", "reports", "cypress", "junit"},
			},
		},
		"configuration": {
			{
				Name:        "Parameterized Workflow",
				Description: "Workflow with configurable parameters",
				YAMLExample: `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: parameterized-test
spec:
  config:
    environment:
      type: string
      default: "staging"
    test_count:
      type: integer
      default: 10
  container:
    image: alpine:3.22.0
  steps:
  - name: Run parameterized test
    shell: |
      echo "Testing in {{ config.environment }} environment"
      echo "Running {{ config.test_count }} tests"
      # Add actual test logic here`,
				UseCases:   []string{"Multi-environment testing", "Configurable test parameters", "Environment-specific tests"},
				Complexity: "intermediate",
				Tags:       []string{"configuration", "parameters", "environment", "flexible"},
			},
		},
	}

	// Get patterns for the specific topic
	patterns, exists := allPatterns[topic]
	if !exists {
		// If topic not found, return general patterns
		patterns = allPatterns[""]
	}

	return patterns
}

// getStepTypes returns step type definitions based on topic
func getStepTypes(topic string) []StepType {
	allStepTypes := map[string][]StepType{
		"": {
			{
				Type:        "shell",
				Description: "Execute shell commands in the container",
				Properties: map[string]Property{
					"shell": {
						Type:        "string",
						Description: "Shell command to execute",
						Required:    true,
						Example:     "echo 'Hello World'",
					},
				},
				Examples: []string{"echo 'Hello World'", "npm test", "python -m pytest"},
				Required: []string{"shell"},
				Optional: []string{"workingDir", "env", "container"},
				YAMLSnippet: `- name: Run command
  shell: echo "Hello World"`,
			},
			{
				Type:        "parallel",
				Description: "Execute steps in parallel with matrix or sharding",
				Properties: map[string]Property{
					"matrix": {
						Type:        "object",
						Description: "Matrix parameters for parallel execution",
						Required:    false,
						Example:     "browser: [chrome, firefox]",
					},
					"count": {
						Type:        "string",
						Description: "Number of parallel instances",
						Required:    false,
						Example:     "5",
					},
					"shards": {
						Type:        "object",
						Description: "Sharding configuration for load distribution",
						Required:    false,
						Example:     "total: 5, current: '{{ matrix.shard }}'",
					},
				},
				Examples: []string{"Matrix testing", "Load distribution", "Parallel execution"},
				Required: []string{},
				Optional: []string{"matrix", "count", "shards", "parallelism"},
				YAMLSnippet: `- name: Parallel execution
  parallel:
    matrix:
      browser: [chrome, firefox]
    shell: echo "Testing {{ matrix.browser }}"`,
			},
		},
		"steps": {
			{
				Type:        "execute",
				Description: "Execute another workflow or test",
				Properties: map[string]Property{
					"workflow": {
						Type:        "string",
						Description: "Name of workflow to execute",
						Required:    false,
						Example:     "my-test-workflow",
					},
					"test": {
						Type:        "string",
						Description: "Name of test to execute",
						Required:    false,
						Example:     "my-test",
					},
					"parallelism": {
						Type:        "integer",
						Description: "Number of parallel executions",
						Required:    false,
						Example:     "3",
					},
				},
				Examples: []string{"Execute workflow", "Run test suite", "Parallel execution"},
				Required: []string{},
				Optional: []string{"workflow", "test", "parallelism", "config"},
				YAMLSnippet: `- name: Execute workflow
  execute:
    workflow: my-test-workflow
    parallelism: 2`,
			},
			{
				Type:        "artifacts",
				Description: "Collect and store artifacts from the execution",
				Properties: map[string]Property{
					"paths": {
						Type:        "array",
						Description: "File patterns to collect",
						Required:    true,
						Example:     "['**/*.xml', 'screenshots/**/*']",
					},
					"workingDir": {
						Type:        "string",
						Description: "Working directory for artifact collection",
						Required:    false,
						Example:     "/data/artifacts",
					},
				},
				Examples: []string{"Collect test reports", "Save screenshots", "Store logs"},
				Required: []string{"paths"},
				Optional: []string{"workingDir"},
				YAMLSnippet: `- name: Collect artifacts
  artifacts:
    paths:
      - "**/*.xml"
      - "screenshots/**/*"`,
			},
		},
		"parallel": {
			{
				Type:        "parallel",
				Description: "Execute steps in parallel with matrix or sharding",
				Properties: map[string]Property{
					"matrix": {
						Type:        "object",
						Description: "Matrix parameters for parallel execution",
						Required:    false,
						Example:     "browser: [chrome, firefox]",
					},
					"count": {
						Type:        "string",
						Description: "Number of parallel instances",
						Required:    false,
						Example:     "5",
					},
					"shards": {
						Type:        "object",
						Description: "Sharding configuration for load distribution",
						Required:    false,
						Example:     "total: 5, current: '{{ matrix.shard }}'",
					},
					"parallelism": {
						Type:        "integer",
						Description: "Maximum number of parallel executions",
						Required:    false,
						Example:     "10",
					},
				},
				Examples: []string{"Matrix testing", "Load distribution", "Parallel execution"},
				Required: []string{},
				Optional: []string{"matrix", "count", "shards", "parallelism"},
				YAMLSnippet: `- name: Parallel execution
  parallel:
    matrix:
      browser: [chrome, firefox]
    shell: echo "Testing {{ matrix.browser }}"`,
			},
		},
		"services": {
			{
				Type:        "services",
				Description: "Define service dependencies for the workflow",
				Properties: map[string]Property{
					"image": {
						Type:        "string",
						Description: "Docker image for the service",
						Required:    true,
						Example:     "postgres:15-alpine",
					},
					"env": {
						Type:        "object",
						Description: "Environment variables for the service",
						Required:    false,
						Example:     "POSTGRES_DB: testdb",
					},
					"readinessProbe": {
						Type:        "object",
						Description: "Health check for service readiness",
						Required:    false,
						Example:     "exec: {command: ['pg_isready']}",
					},
				},
				Examples: []string{"Database services", "API services", "Message queues"},
				Required: []string{"image"},
				Optional: []string{"env", "readinessProbe", "ports"},
				YAMLSnippet: `services:
  postgres:
    image: postgres:15-alpine
    env:
      POSTGRES_DB: testdb
    readinessProbe:
      exec:
        command: ["pg_isready"]`,
			},
		},
	}

	// Get step types for the specific topic
	stepTypes, exists := allStepTypes[topic]
	if !exists {
		// If topic not found, return general step types
		stepTypes = allStepTypes[""]
	}

	return stepTypes
}

// getBestPractices returns best practices based on topic
func getBestPractices(topic string) []BestPractice {
	allBestPractices := map[string][]BestPractice{
		"": {
			{
				Category:     "Resource Management",
				Rule:         "Set appropriate resource limits",
				Description:  "Always specify CPU and memory limits to prevent resource exhaustion",
				Examples:     []string{"cpu: 500m, memory: 1Gi", "cpu: 100m, memory: 256Mi"},
				AntiPatterns: []string{"No resource limits", "Excessive resource requests"},
			},
			{
				Category:     "Error Handling",
				Rule:         "Use retry policies for flaky operations",
				Description:  "Implement retry logic for network calls and external dependencies",
				Examples:     []string{"retry: {count: 3, until: 'passed'}", "condition: 'always' for cleanup"},
				AntiPatterns: []string{"No retry logic", "Infinite retries"},
			},
		},
		"parallel": {
			{
				Category:     "Performance",
				Rule:         "Limit parallelism based on available resources",
				Description:  "Don't exceed cluster capacity with too many parallel executions",
				Examples:     []string{"parallelism: 5", "maxCount: 10"},
				AntiPatterns: []string{"Unlimited parallelism", "No resource consideration"},
			},
		},
		"services": {
			{
				Category:     "Dependencies",
				Rule:         "Always use readiness probes for services",
				Description:  "Ensure services are ready before running dependent steps",
				Examples:     []string{"readinessProbe with health checks", "Wait loops with proper conditions"},
				AntiPatterns: []string{"No health checks", "Immediate execution after service start"},
			},
		},
	}

	// Get best practices for the specific topic
	bestPractices, exists := allBestPractices[topic]
	if !exists {
		// If topic not found, return general best practices
		bestPractices = allBestPractices[""]
	}

	return bestPractices
}

// getQuickStartGuide returns quick start steps based on topic
func getQuickStartGuide(topic string) []string {
	guides := map[string][]string{
		"": {
			"🎯 **QUICK START GUIDE** - Follow these steps to build your first TestWorkflow:",
			"",
			"1. Start with a simple shell step to test your environment",
			"2. Add resource limits to prevent resource exhaustion",
			"3. Use conditional steps for error handling",
			"4. Collect artifacts for test reports",
			"5. Test your workflow with a simple example first",
			"",
			"💡 **Tip**: Use this guide when creating your first workflow or when you're unsure where to start!",
		},
		"steps": {
			"🎯 **STEPS QUICK START** - Building effective workflow steps:",
			"",
			"1. Begin with basic shell execution",
			"2. Add conditional logic for error handling",
			"3. Implement retry policies for flaky operations",
			"4. Use artifacts collection for reports",
			"5. Test each step individually before combining",
			"",
			"💡 **Tip**: Perfect for learning step configuration and error handling patterns!",
		},
		"parallel": {
			"🎯 **PARALLEL EXECUTION QUICK START** - Scaling your workflows:",
			"",
			"1. Start with simple parallel execution (count: 2)",
			"2. Add matrix parameters for different configurations",
			"3. Use sharding for load distribution",
			"4. Monitor resource usage with high parallelism",
			"5. Test with small datasets first",
			"",
			"💡 **Tip**: Use this when you need to run tests across multiple environments or configurations!",
		},
		"services": {
			"🎯 **SERVICES QUICK START** - Adding service dependencies:",
			"",
			"1. Define service with proper health checks",
			"2. Add readiness probes for dependencies",
			"3. Wait for services to be ready before testing",
			"4. Use environment variables for service configuration",
			"5. Test service connectivity before complex operations",
			"",
			"💡 **Tip**: Essential for integration testing with databases, APIs, or other services!",
		},
		"artifacts": {
			"🎯 **ARTIFACTS QUICK START** - Collecting test outputs:",
			"",
			"1. Identify what files you need to collect (reports, logs, screenshots)",
			"2. Set up proper file patterns for collection",
			"3. Configure working directory for artifact storage",
			"4. Test artifact collection with a simple example",
			"5. Verify artifacts are accessible after execution",
			"",
			"💡 **Tip**: Use this when you need to preserve test results, screenshots, or reports!",
		},
		"configuration": {
			"🎯 **CONFIGURATION QUICK START** - Making workflows flexible:",
			"",
			"1. Identify parameters that should be configurable",
			"2. Define config schema with types and defaults",
			"3. Use config variables in your workflow steps",
			"4. Test with different parameter values",
			"5. Document expected parameter values",
			"",
			"💡 **Tip**: Perfect for creating reusable workflows across different environments!",
		},
	}

	guide, exists := guides[topic]
	if !exists {
		guide = guides[""]
	}

	return guide
}

// getCommonMistakes returns common mistakes based on topic
func getCommonMistakes(topic string) []string {
	mistakes := map[string][]string{
		"": {
			"Not setting resource limits leading to cluster resource exhaustion",
			"Using 'latest' tags for images instead of specific versions",
			"Not handling errors with conditional steps or retry policies",
			"Missing artifact collection for important test outputs",
			"Not testing workflows locally before deployment",
		},
		"parallel": {
			"Setting parallelism too high without considering cluster capacity",
			"Not using matrix parameters when testing multiple configurations",
			"Forgetting to handle matrix variable references in shell commands",
			"Not monitoring resource usage during parallel execution",
		},
		"services": {
			"Not waiting for services to be ready before running tests",
			"Missing health checks or readiness probes",
			"Not handling service startup failures gracefully",
			"Using hardcoded service URLs instead of service names",
		},
		"artifacts": {
			"Not specifying correct file patterns for artifact collection",
			"Collecting too many files leading to storage issues",
			"Not setting proper working directory for artifact collection",
			"Forgetting to collect important test reports",
		},
	}

	mistakeList, exists := mistakes[topic]
	if !exists {
		mistakeList = mistakes[""]
	}

	return mistakeList
}

// formatEnhancedDocumentationResponse formats the enhanced response as readable text
func formatEnhancedDocumentationResponse(response EnhancedDocumentationResponse) string {
	var result strings.Builder

	result.WriteString("# TestWorkflow Documentation Guide\n\n")

	// Add usage instructions for LLMs
	result.WriteString("> **🤖 For LLMs**: This documentation provides comprehensive guidance for creating TestWorkflows. Here's how to use it effectively:\n")
	result.WriteString("> - **Quick Start Guide**: Follow step-by-step instructions for building workflows\n")
	result.WriteString("> - **Workflow Patterns**: Use complete YAML examples as templates\n")
	result.WriteString("> - **Step Types Reference**: Check property requirements and examples\n")
	result.WriteString("> - **Best Practices**: Follow recommended patterns and avoid common mistakes\n")
	result.WriteString("> - **Common Mistakes**: Learn what to avoid before you make errors\n\n")

	if response.Topic != "" {
		result.WriteString(fmt.Sprintf("## Topic: %s\n\n", strings.Title(response.Topic)))
	}

	// Documentation Pages
	if len(response.Pages) > 0 {
		result.WriteString("## 📚 Documentation Pages\n\n")
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

	// Workflow Patterns
	if len(response.Patterns) > 0 {
		result.WriteString("## 🔧 Workflow Patterns\n\n")
		for i, pattern := range response.Patterns {
			result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, pattern.Name))
			result.WriteString(fmt.Sprintf("**Description:** %s\n\n", pattern.Description))
			result.WriteString(fmt.Sprintf("**Complexity:** %s\n\n", pattern.Complexity))
			if len(pattern.UseCases) > 0 {
				result.WriteString("**Use Cases:** ")
				result.WriteString(strings.Join(pattern.UseCases, ", "))
				result.WriteString("\n\n")
			}
			if len(pattern.Tags) > 0 {
				result.WriteString("**Tags:** ")
				result.WriteString(strings.Join(pattern.Tags, ", "))
				result.WriteString("\n\n")
			}
			result.WriteString("**YAML Example:**\n```yaml\n")
			result.WriteString(pattern.YAMLExample)
			result.WriteString("\n```\n\n")
			result.WriteString("---\n\n")
		}
	}

	// Step Types
	if len(response.StepTypes) > 0 {
		result.WriteString("## ⚙️ Step Types Reference\n\n")
		for i, stepType := range response.StepTypes {
			result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, stepType.Type))
			result.WriteString(fmt.Sprintf("**Description:** %s\n\n", stepType.Description))

			if len(stepType.Properties) > 0 {
				result.WriteString("**Properties:**\n")
				for propName, prop := range stepType.Properties {
					required := ""
					if prop.Required {
						required = " (required)"
					}
					result.WriteString(fmt.Sprintf("- **%s** (%s)%s: %s\n", propName, prop.Type, required, prop.Description))
					if prop.Example != "" {
						result.WriteString(fmt.Sprintf("  - Example: `%s`\n", prop.Example))
					}
				}
				result.WriteString("\n")
			}

			if len(stepType.Examples) > 0 {
				result.WriteString("**Examples:** ")
				result.WriteString(strings.Join(stepType.Examples, ", "))
				result.WriteString("\n\n")
			}

			if stepType.YAMLSnippet != "" {
				result.WriteString("**YAML Snippet:**\n```yaml\n")
				result.WriteString(stepType.YAMLSnippet)
				result.WriteString("\n```\n\n")
			}
			result.WriteString("---\n\n")
		}
	}

	// Best Practices
	if len(response.BestPractices) > 0 {
		result.WriteString("## ✅ Best Practices\n\n")
		for i, practice := range response.BestPractices {
			result.WriteString(fmt.Sprintf("### %d. %s - %s\n", i+1, practice.Category, practice.Rule))
			result.WriteString(fmt.Sprintf("**Description:** %s\n\n", practice.Description))

			if len(practice.Examples) > 0 {
				result.WriteString("**Examples:**\n")
				for _, example := range practice.Examples {
					result.WriteString(fmt.Sprintf("- %s\n", example))
				}
				result.WriteString("\n")
			}

			if len(practice.AntiPatterns) > 0 {
				result.WriteString("**Anti-patterns to avoid:**\n")
				for _, antiPattern := range practice.AntiPatterns {
					result.WriteString(fmt.Sprintf("- ❌ %s\n", antiPattern))
				}
				result.WriteString("\n")
			}
			result.WriteString("---\n\n")
		}
	}

	// Quick Start Guide
	if len(response.QuickStart) > 0 {
		result.WriteString("## 🚀 Quick Start Guide\n\n")
		result.WriteString("> **💡 For LLMs**: This guide provides step-by-step instructions for building TestWorkflows. Follow these steps when you're creating a new workflow or learning a specific topic. Each step builds on the previous one, so follow them in order for best results.\n\n")
		for _, step := range response.QuickStart {
			result.WriteString(fmt.Sprintf("%s\n", step))
		}
		result.WriteString("\n")
	}

	// Common Mistakes
	if len(response.CommonMistakes) > 0 {
		result.WriteString("## ⚠️ Common Mistakes to Avoid\n\n")
		for i, mistake := range response.CommonMistakes {
			result.WriteString(fmt.Sprintf("%d. ❌ %s\n", i+1, mistake))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// formatDocumentationResponse formats the response as a readable text (legacy function)
func formatDocumentationResponse(response EnhancedDocumentationResponse) string {
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
