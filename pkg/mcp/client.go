package mcp

import "github.com/kubeshop/testkube/pkg/mcp/tools"

type Client interface {
	tools.ArtifactLister
	tools.ArtifactReader

	tools.ExecutionLogger
	tools.ExecutionLister
	tools.ExecutionInfoGetter
	tools.ExecutionLookup
	tools.ExecutionWaiter
	tools.WorkflowExecutionAborter
	tools.WorkflowExecutionMetricsGetter
	tools.WorkflowResourceHistoryGetter

	tools.LabelsLister
	tools.ResourceGroupsLister
	tools.AgentsLister

	tools.WorkflowLister
	tools.WorkflowCreator
	tools.WorkflowUpdater
	tools.WorkflowDefinitionGetter
	tools.WorkflowGetter
	tools.WorkflowRunner
	tools.WorkflowMetricsGetter

	// WorkflowTemplate interfaces
	tools.WorkflowTemplateLister
	tools.WorkflowTemplateDefinitionGetter
	tools.WorkflowTemplateCreator
	tools.WorkflowTemplateUpdater

	// Bulk getters for yq query tools
	tools.WorkflowDefinitionBulkGetter
	tools.ExecutionBulkGetter
}
