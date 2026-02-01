package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	logging "gopkg.in/op/go-logging.v1"
	"gopkg.in/yaml.v3"
)

func init() {
	// Disable verbose yq debug logging
	logging.SetLevel(logging.ERROR, "yq-lib")

	// SECURITY: Disable dangerous yq operators that could leak secrets or read files
	// - env(): reads environment variables (could expose API tokens, secrets)
	// - envsubst(): substitutes ${VAR} from environment
	// - load()/strload(): reads arbitrary files from filesystem
	yqlib.ConfiguredSecurityPreferences.DisableEnvOps = true
	yqlib.ConfiguredSecurityPreferences.DisableFileOps = true
}

const (
	// Maximum output size for yq queries (100KB)
	maxOutputSize = 100 * 1024

	// Default timeout for yq expression evaluation
	defaultYqTimeout = 10 * time.Second

	// Default limit for bulk fetches
	defaultBulkLimit = 50

	// Maximum expression length
	maxExpressionLength = 10000

	// Maximum input size (10MB)
	maxInputSize = 10 * 1024 * 1024
)

// blockedOperatorPatterns contains regex patterns for operators that should be blocked
// even though yq's security preferences might not cover them
var blockedOperatorPatterns = regexp.MustCompile(`(?i)\b(env|envsubst|load|strload)\s*\(`)

// validateExpression checks the expression for potentially dangerous patterns
func validateExpression(expression string) error {
	if len(expression) > maxExpressionLength {
		return fmt.Errorf("expression too long: %d characters (max %d)", len(expression), maxExpressionLength)
	}

	// Double-check for blocked operators (defense in depth)
	if blockedOperatorPatterns.MatchString(expression) {
		return fmt.Errorf("expression contains blocked operator: env, envsubst, load, or strload are not allowed")
	}

	return nil
}

// WorkflowDefinitionBulkGetter retrieves multiple workflow definitions in bulk
type WorkflowDefinitionBulkGetter interface {
	GetWorkflowDefinitions(ctx context.Context, params ListWorkflowsParams) (map[string]string, error)
}

// ExecutionBulkGetter retrieves multiple execution records in bulk
type ExecutionBulkGetter interface {
	GetExecutions(ctx context.Context, params ListExecutionsParams) (map[string]string, error)
}

// executeYqQuery applies a yq expression to the input and returns the result
// Security measures:
// - Expression validation (blocks dangerous operators)
// - Timeout enforcement (default 10s)
// - Panic recovery to prevent crashes
// - Output size limit (100KB)
// - Input size limit (10MB)
// - Dangerous operators disabled (env, load) via init()
func executeYqQuery(expression string, input string, isYaml bool, timeout time.Duration) (string, error) {
	if timeout == 0 {
		timeout = defaultYqTimeout
	}

	// Validate expression for dangerous patterns
	if err := validateExpression(expression); err != nil {
		return "", err
	}

	// Validate input size to prevent memory exhaustion
	if len(input) > maxInputSize {
		return "", fmt.Errorf("input too large: %d bytes (max %d)", len(input), maxInputSize)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel to receive result
	resultCh := make(chan struct {
		output string
		err    error
	}, 1)

	go func() {
		// Recover from any panics in the yq library
		defer func() {
			if r := recover(); r != nil {
				resultCh <- struct {
					output string
					err    error
				}{"", fmt.Errorf("yq query panicked: %v", r)}
			}
		}()

		output, err := runYqExpression(expression, input, isYaml)
		resultCh <- struct {
			output string
			err    error
		}{output, err}
	}()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("yq query timed out after %v", timeout)
	case result := <-resultCh:
		if result.err != nil {
			return "", result.err
		}
		if len(result.output) > maxOutputSize {
			return "", fmt.Errorf("output exceeds maximum size of %d bytes (got %d bytes)", maxOutputSize, len(result.output))
		}
		return result.output, nil
	}
}

// runYqExpression executes the yq expression against the input
func runYqExpression(expression string, input string, isYaml bool) (string, error) {
	// Parse the input
	var node yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(input))
	if err := decoder.Decode(&node); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	// Create the yq evaluator
	evaluator := yqlib.NewStringEvaluator()

	// Configure encoder preferences for readable output
	prefs := yqlib.ConfiguredYamlPreferences.Copy()
	prefs.Indent = 2

	// Create encoder
	encoder := yqlib.NewYamlEncoder(prefs)

	// Create decoder based on input type
	var inputDecoder yqlib.Decoder
	if isYaml {
		inputDecoder = yqlib.NewYamlDecoder(yqlib.ConfiguredYamlPreferences)
	} else {
		inputDecoder = yqlib.NewJSONDecoder()
	}

	// Evaluate the expression
	result, err := evaluator.EvaluateAll(expression, input, encoder, inputDecoder)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate yq expression: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// QueryWorkflowsYq creates a tool for querying workflow definitions with yq
func QueryWorkflowsYq(client WorkflowDefinitionBulkGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("query_workflows_yq",
		mcp.WithDescription(QueryWorkflowsYqDescription),
		mcp.WithString("expression", mcp.Required(), mcp.Description("The yq expression to apply to workflow definitions. Examples: '.spec.steps[].container.image', '.spec.services | keys', 'select(.spec.steps[].container.image | contains(\"python\"))'.")),
		mcp.WithString("selector", mcp.Description(SelectorDescription)),
		mcp.WithString("resourceGroup", mcp.Description(ResourceGroupDescription)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of workflows to fetch (default: 50, max: 100).")),
		mcp.WithBoolean("aggregate", mcp.Description("If true, combines all workflows into a multi-document YAML and applies the expression once. If false (default), applies the expression to each workflow separately.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		expression, err := RequiredParam[string](request, "expression")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		selector, _ := OptionalParam[string](request, "selector")
		resourceGroup, _ := OptionalParam[string](request, "resourceGroup")
		limit, _ := OptionalIntParamWithDefault(request, "limit", defaultBulkLimit)
		aggregate, _ := OptionalParam[bool](request, "aggregate")

		// Cap limit at 100
		if limit > 100 {
			limit = 100
		}

		params := ListWorkflowsParams{
			Selector:      selector,
			ResourceGroup: resourceGroup,
			PageSize:      limit,
			Page:          0,
		}

		// Fetch workflow definitions
		definitions, err := client.GetWorkflowDefinitions(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch workflow definitions: %v", err)), nil
		}

		if len(definitions) == 0 {
			return mcp.NewToolResultText("No workflows found matching the criteria."), nil
		}

		var result string
		if aggregate {
			// Combine all workflows into multi-document YAML
			var combined strings.Builder
			first := true
			for _, def := range definitions {
				if !first {
					combined.WriteString("---\n")
				}
				combined.WriteString(def)
				combined.WriteString("\n")
				first = false
			}

			// Apply expression once to combined document
			output, err := executeYqQuery(expression, combined.String(), true, defaultYqTimeout)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to execute yq query: %v", err)), nil
			}
			result = output
		} else {
			// Apply expression to each workflow separately
			results := make(map[string]string)
			for name, def := range definitions {
				output, err := executeYqQuery(expression, def, true, defaultYqTimeout)
				if err != nil {
					results[name] = fmt.Sprintf("ERROR: %v", err)
				} else {
					results[name] = output
				}
			}

			// Format as YAML for readability
			resultBytes, err := yaml.Marshal(results)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
			}
			result = string(resultBytes)
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

// QueryExecutionsYq creates a tool for querying execution data with yq
func QueryExecutionsYq(client ExecutionBulkGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("query_executions_yq",
		mcp.WithDescription(QueryExecutionsYqDescription),
		mcp.WithString("expression", mcp.Required(), mcp.Description("The yq expression to apply to execution data. Examples: '.result.status', '.result.duration', '.result.steps[] | select(.result.status == \"failed\") | .name'.")),
		mcp.WithString("workflowName", mcp.Description(WorkflowNameDescription)),
		mcp.WithString("status", mcp.Description(StatusDescription)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of executions to fetch (default: 50, max: 100).")),
		mcp.WithBoolean("aggregate", mcp.Description("If true, combines all executions into a JSON array and applies the expression once. If false (default), applies the expression to each execution separately.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		expression, err := RequiredParam[string](request, "expression")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		workflowName, _ := OptionalParam[string](request, "workflowName")
		status, _ := OptionalParam[string](request, "status")
		limit, _ := OptionalIntParamWithDefault(request, "limit", defaultBulkLimit)
		aggregate, _ := OptionalParam[bool](request, "aggregate")

		// Cap limit at 100
		if limit > 100 {
			limit = 100
		}

		params := ListExecutionsParams{
			WorkflowName: workflowName,
			Status:       status,
			PageSize:     limit,
			Page:         0,
		}

		// Fetch executions
		executions, err := client.GetExecutions(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch executions: %v", err)), nil
		}

		if len(executions) == 0 {
			return mcp.NewToolResultText("No executions found matching the criteria."), nil
		}

		var result string
		if aggregate {
			// Combine all executions into a JSON array
			var items []json.RawMessage
			for _, exec := range executions {
				items = append(items, json.RawMessage(exec))
			}

			combinedBytes, err := json.Marshal(items)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to combine executions: %v", err)), nil
			}

			// Apply expression once to combined array
			output, err := executeYqQuery(expression, string(combinedBytes), false, defaultYqTimeout)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to execute yq query: %v", err)), nil
			}
			result = output
		} else {
			// Apply expression to each execution separately
			results := make(map[string]string)
			for id, exec := range executions {
				output, err := executeYqQuery(expression, exec, false, defaultYqTimeout)
				if err != nil {
					results[id] = fmt.Sprintf("ERROR: %v", err)
				} else {
					results[id] = output
				}
			}

			// Format as YAML for readability
			resultBytes, err := yaml.Marshal(results)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
			}
			result = string(resultBytes)
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}
