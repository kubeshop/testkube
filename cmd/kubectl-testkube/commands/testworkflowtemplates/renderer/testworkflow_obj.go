package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestWorkflowTemplateRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
	template, ok := obj.(testkube.TestWorkflowTemplate)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestWorkflowTemplate in RenderObj for test workflow template", obj)
	}

	ui.Info("Test Workflow Template:")
	ui.Warn("Name:     ", template.Name)
	ui.Warn("Namespace:", template.Namespace)
	ui.Warn("Created:  ", template.Created.String())
	if template.Description != "" {
		ui.NL()
		ui.Warn("Description: ", template.Description)
	}
	if len(template.Labels) > 0 {
		ui.NL()
		ui.Warn("Labels:   ", testkube.MapToString(template.Labels))
	}

	return nil

}
