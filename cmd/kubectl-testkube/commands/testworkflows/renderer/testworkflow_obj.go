package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestWorkflowRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
	workflow, ok := obj.(testkube.TestWorkflow)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestWorkflow in RenderObj for test workflow", obj)
	}

	ui.Info("Test Workflow:")
	ui.Warn("Name:     ", workflow.Name)
	ui.Warn("Namespace:", workflow.Namespace)
	ui.Warn("Created:  ", workflow.Created.String())
	if workflow.Description != "" {
		ui.NL()
		ui.Warn("Description: ", workflow.Description)
	}
	if len(workflow.Labels) > 0 {
		ui.NL()
		ui.Warn("Labels:   ", testkube.MapToString(workflow.Labels))
	}

	return nil

}
