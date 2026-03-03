package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/formatters"
)

type WorkflowTemplateLister interface {
	ListWorkflowTemplates(ctx context.Context, selector string) (string, error)
}

func ListWorkflowTemplates(client WorkflowTemplateLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_workflowtemplates",
		mcp.WithDescription(ListWorkflowTemplatesDescription),
		mcp.WithString("selector", mcp.Description(SelectorDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		selector := request.GetString("selector", "")

		result, err := client.ListWorkflowTemplates(ctx, selector)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list workflow templates: %v", err)), nil
		}

		formatted, err := formatters.FormatListWorkflowTemplates(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format workflow templates: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

type WorkflowTemplateDefinitionGetter interface {
	GetWorkflowTemplateDefinition(ctx context.Context, templateName string) (string, error)
}

func GetWorkflowTemplateDefinition(client WorkflowTemplateDefinitionGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflowtemplate_definition",
		mcp.WithDescription(GetWorkflowTemplateDefinitionDescription),
		mcp.WithString("templateName", mcp.Required(), mcp.Description(TemplateNameDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateName, err := RequiredParam[string](request, "templateName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.GetWorkflowTemplateDefinition(ctx, templateName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get workflow template definition: %v", err)), nil
		}

		formatted, err := formatters.FormatGetWorkflowTemplateDefinition(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format workflow template definition: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

type WorkflowTemplateCreator interface {
	CreateWorkflowTemplate(ctx context.Context, templateDefinition string) (string, error)
}

func CreateWorkflowTemplate(client WorkflowTemplateCreator) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("create_workflowtemplate",
		mcp.WithDescription(CreateWorkflowTemplateDescription),
		mcp.WithString("yaml", mcp.Required(), mcp.Description("Complete YAML definition of the TestWorkflowTemplate to create in Testkube. This should be the full template specification including metadata and spec.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		yaml, err := RequiredParam[string](request, "yaml")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.CreateWorkflowTemplate(ctx, yaml)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create workflow template: %v", err)), nil
		}

		formatted, err := formatters.FormatCreateWorkflowTemplate(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format workflow template: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

type WorkflowTemplateUpdater interface {
	UpdateWorkflowTemplate(ctx context.Context, templateName, templateDefinition string) (string, error)
}

func UpdateWorkflowTemplate(client WorkflowTemplateUpdater) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("update_workflowtemplate",
		mcp.WithDescription(UpdateWorkflowTemplateDescription),
		mcp.WithString("templateName", mcp.Required(), mcp.Description(TemplateNameDescription)),
		mcp.WithString("yaml", mcp.Required(), mcp.Description("Complete YAML definition of the TestWorkflowTemplate to update in Testkube. This should be the full template specification including metadata and spec.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateName, err := RequiredParam[string](request, "templateName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		yaml, err := RequiredParam[string](request, "yaml")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.UpdateWorkflowTemplate(ctx, templateName, yaml)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update workflow template: %v", err)), nil
		}

		formatted, err := formatters.FormatUpdateWorkflowTemplate(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format workflow template: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}
