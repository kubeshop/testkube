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
}
