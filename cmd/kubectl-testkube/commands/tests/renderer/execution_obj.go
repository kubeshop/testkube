package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func ExecutionRenderer(ui *ui.UI, obj interface{}) error {
	execution, ok := obj.(testkube.Execution)
	if !ok {
		return fmt.Errorf("can't render execution, expecrted obj to be testkube.Execution but got '%T'", obj)
	}

	ui.Warn("ID:       ", execution.Id)
	ui.Warn("Name:     ", execution.Name)
	ui.Warn("Type:     ", execution.TestType)
	ui.Warn("Duration: ", execution.Duration)

	if len(execution.Labels) > 0 {
		ui.Warn("Labels:   ", testkube.MapToString(execution.Labels))
	}

	renderer.RenderVariables(execution.Variables)

	if len(execution.Args) > 0 {
		ui.Warn("Args:    ", execution.Args...)
	}

	renderer.RenderExecutionResult(execution.ExecutionResult)

	ui.NL()

	return nil
}
