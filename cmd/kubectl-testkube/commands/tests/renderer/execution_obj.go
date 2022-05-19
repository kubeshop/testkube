package renderer

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func ExecutionRenderer(ui *ui.UI, obj interface{}) error {
	execution, ok := obj.(testkube.Execution)
	if !ok {
		return fmt.Errorf("can't render execution, expecrted obj to be testkube.Execution but got '%T'", obj)
	}
	result := execution.ExecutionResult

	ui.Warn("ID:       ", execution.Id)
	ui.Warn("Name:     ", execution.Name)
	ui.Warn("Type:     ", execution.TestType)
	ui.Warn("Duration: ", execution.Duration)

	if len(execution.Labels) > 0 {
		ui.Warn("Labels:   ", testkube.LabelsToString(execution.Labels))
	}

	renderer.RenderVariables(execution.Variables)

	if len(execution.Args) > 0 {
		ui.Warn("Args:    ", execution.Args...)
	}

	if result == nil {
		return fmt.Errorf("got execution without `Result`")
	}

	ui.NL()

	switch true {
	case result.IsQueued():
		ui.Warn("Status", "test queued for execution")

	case result.IsRunning():
		ui.Warn("Test execution started")

	case result.IsPassed():
		ui.Info(result.Output)
		duration := execution.EndTime.Sub(execution.StartTime)
		ui.Success("Status", "Test execution completed with success in "+duration.String())

	case result.IsFailed():
		ui.Warn("Status", "test execution failed:\n")
		ui.Errf(result.ErrorMessage)
		ui.Info(result.Output)
		os.Exit(1)

	default:
		ui.Warn("Status", "test execution status unknown:\n")
		ui.Errf(result.ErrorMessage)
		ui.Info(result.Output)
		os.Exit(1)
	}

	ui.NL()

	return nil
}
