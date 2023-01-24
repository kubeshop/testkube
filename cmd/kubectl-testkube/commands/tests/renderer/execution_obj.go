package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func ExecutionRenderer(ui *ui.UI, obj interface{}) error {
	execution, ok := obj.(testkube.Execution)
	if !ok {
		return fmt.Errorf("can't render execution, expecrted obj to be testkube.Execution but got '%T'", obj)
	}

	ui.Warn("ID:        ", execution.Id)
	ui.Warn("Name:      ", execution.Name)
	if execution.Number != 0 {
		ui.Warn("Number:           ", fmt.Sprintf("%d", execution.Number))
	}
	ui.Warn("Test name:        ", execution.TestName)
	ui.Warn("Type:             ", execution.TestType)
	ui.Warn("Status:           ", string(*execution.ExecutionResult.Status))
	ui.Warn("Start time:       ", execution.StartTime.String())
	ui.Warn("End time:         ", execution.EndTime.String())
	ui.Warn("Duration:         ", execution.Duration)

	if len(execution.Labels) > 0 {
		ui.Warn("Labels:           ", testkube.MapToString(execution.Labels))
	}

	renderer.RenderVariables(execution.Variables)

	if len(execution.Args) > 0 {
		ui.Warn("Args:    ", execution.Args...)
	}

	if execution.Content != nil && execution.Content.Repository != nil {
		ui.Warn("Repository parameters:")
		ui.Warn("  Branch:         ", execution.Content.Repository.Branch)
		ui.Warn("  Commit:         ", execution.Content.Repository.Commit)
		ui.Warn("  Path:           ", execution.Content.Repository.Path)
		ui.Warn("  Working dir:    ", execution.Content.Repository.WorkingDir)
		ui.Warn("  Certificate:    ", execution.Content.Repository.CertificateSecret)
	}

	render.RenderExecutionResult(&execution)

	ui.NL()

	return nil
}
