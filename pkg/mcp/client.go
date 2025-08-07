package mcp

import "github.com/kubeshop/testkube/pkg/mcp/tools"

type Client interface {
	tools.ArtifactLister
	tools.ArtifactReader

	tools.ExecutionLogger
	tools.ExecutionLister
	tools.ExecutionInfoGetter
	tools.ExecutionLookup

	tools.LabelsLister

	tools.WorkflowLister
	tools.WorkflowCreator
	tools.WorkflowDefinitionGetter
	tools.WorkflowGetter
	tools.WorkflowRunner
}
