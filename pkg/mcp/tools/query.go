package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/testkube/pkg/mcp/jsonpath"
)

const (
	// defaultBulkLimit is the default number of items to fetch
	defaultBulkLimit = 50
)

// WorkflowDefinitionBulkGetter retrieves multiple workflow definitions in bulk
type WorkflowDefinitionBulkGetter interface {
	GetWorkflowDefinitions(ctx context.Context, params ListWorkflowsParams) (map[string]string, error)
}

// ExecutionBulkGetter retrieves multiple execution records in bulk
type ExecutionBulkGetter interface {
	GetExecutions(ctx context.Context, params ListExecutionsParams) (map[string]string, error)
}

// QueryWorkflows creates a tool for querying workflow definitions with JSONPath
func QueryWorkflows(client WorkflowDefinitionBulkGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("query_workflows",
		mcp.WithDescription(QueryWorkflowsDescription),
		mcp.WithString("expression", mcp.Required(), mcp.Description("The JSONPath expression to apply to workflow definitions. Examples: '$..image', '$.spec.steps[*].name', '$.metadata.labels'.")),
		mcp.WithString("selector", mcp.Description(SelectorDescription)),
		mcp.WithString("resourceGroup", mcp.Description(ResourceGroupDescription)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of workflows to fetch (default: 50, max: 100).")),
		mcp.WithBoolean("aggregate", mcp.Description("If true, combines all workflows into an array and applies the expression once. If false (default), applies the expression to each workflow separately.")),
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
			// Combine all workflows into an array and query once
			var allWorkflows []any
			for _, def := range definitions {
				var parsed any
				if err := yaml.Unmarshal([]byte(def), &parsed); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to parse workflow YAML: %v", err)), nil
				}
				allWorkflows = append(allWorkflows, parsed)
			}

			// Apply expression to the combined array
			queryResult, err := jsonpath.QueryWithContext(ctx, expression, allWorkflows, jsonpath.DefaultOptions())
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to execute JSONPath query: %v", err)), nil
			}

			// Format result as JSON
			resultBytes, err := json.MarshalIndent(queryResult, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
			}
			result = string(resultBytes)
		} else {
			// Apply expression to each workflow separately
			results := make(map[string]any)
			for name, def := range definitions {
				var parsed any
				if err := yaml.Unmarshal([]byte(def), &parsed); err != nil {
					results[name] = fmt.Sprintf("ERROR: failed to parse YAML: %v", err)
					continue
				}

				queryResult, err := jsonpath.QueryWithContext(ctx, expression, parsed, jsonpath.DefaultOptions())
				if err != nil {
					results[name] = fmt.Sprintf("ERROR: %v", err)
				} else {
					results[name] = queryResult
				}
			}

			// Format as JSON for readability
			resultBytes, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
			}
			result = string(resultBytes)
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

// QueryExecutions creates a tool for querying execution data with JSONPath
func QueryExecutions(client ExecutionBulkGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("query_executions",
		mcp.WithDescription(QueryExecutionsDescription),
		mcp.WithString("expression", mcp.Required(), mcp.Description("The JSONPath expression to apply to execution data. Examples: '$.result.status', '$.result.duration', '$..errorMessage'.")),
		mcp.WithString("workflowName", mcp.Description(WorkflowNameDescription)),
		mcp.WithString("status", mcp.Description(StatusDescription)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of executions to fetch (default: 50, max: 100).")),
		mcp.WithBoolean("aggregate", mcp.Description("If true, combines all executions into an array and applies the expression once. If false (default), applies the expression to each execution separately.")),
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
			// Combine all executions into an array and query once
			var allExecutions []any
			for _, exec := range executions {
				var parsed any
				if err := json.Unmarshal([]byte(exec), &parsed); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to parse execution JSON: %v", err)), nil
				}
				allExecutions = append(allExecutions, parsed)
			}

			// Apply expression to the combined array
			queryResult, err := jsonpath.QueryWithContext(ctx, expression, allExecutions, jsonpath.DefaultOptions())
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to execute JSONPath query: %v", err)), nil
			}

			// Format result as JSON
			resultBytes, err := json.MarshalIndent(queryResult, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
			}
			result = string(resultBytes)
		} else {
			// Apply expression to each execution separately
			results := make(map[string]any)
			for id, exec := range executions {
				var parsed any
				if err := json.Unmarshal([]byte(exec), &parsed); err != nil {
					results[id] = fmt.Sprintf("ERROR: failed to parse JSON: %v", err)
					continue
				}

				queryResult, err := jsonpath.QueryWithContext(ctx, expression, parsed, jsonpath.DefaultOptions())
				if err != nil {
					results[id] = fmt.Sprintf("ERROR: %v", err)
				} else {
					results[id] = queryResult
				}
			}

			// Format as JSON for readability
			resultBytes, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
			}
			result = string(resultBytes)
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}
