package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func ExecutionRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
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
	ui.Warn("Test namespace:   ", execution.TestNamespace)
	ui.Warn("Type:             ", execution.TestType)
	if execution.ExecutionResult != nil && execution.ExecutionResult.Status != nil {
		ui.Warn("Status:           ", string(*execution.ExecutionResult.Status))
	}
	ui.Warn("Start time:       ", execution.StartTime.String())
	ui.Warn("End time:         ", execution.EndTime.String())
	ui.Warn("Duration:         ", execution.Duration)
	if execution.RunningContext != nil {
		ui.Warn("Running context:")
		ui.Warn("Type:   ", execution.RunningContext.Type_)
		ui.Warn("Context:", execution.RunningContext.Context)
	}

	if len(execution.Labels) > 0 {
		ui.Warn("Labels:           ", testkube.MapToString(execution.Labels))
	}

	renderer.RenderVariables(execution.Variables)

	if len(execution.Command) > 0 {
		ui.Warn("Command:          ", execution.Command...)
	}

	if len(execution.Args) > 0 {
		ui.Warn("Args:             ", execution.Args...)
	}

	if execution.Content != nil && execution.Content.Repository != nil {
		ui.Warn("Repository parameters:")
		ui.Warn("  Branch:         ", execution.Content.Repository.Branch)
		ui.Warn("  Commit:         ", execution.Content.Repository.Commit)
		ui.Warn("  Path:           ", execution.Content.Repository.Path)
		ui.Warn("  Working dir:    ", execution.Content.Repository.WorkingDir)
		ui.Warn("  Certificate:    ", execution.Content.Repository.CertificateSecret)
		ui.Warn("  Auth type:      ", execution.Content.Repository.AuthType)
	}

	if err := render.RenderExecutionResult(client, &execution, false, true); err != nil {
		return err
	}

	ui.NL()

	return nil
}
