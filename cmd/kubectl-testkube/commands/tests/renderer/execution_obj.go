package renderer

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func ExecutionRenderer(ui *ui.UI, obj interface{}) error {
	execution, ok := obj.(testkube.Execution)
	if !ok {
		return fmt.Errorf("can't render execution, expecrted obj to be testkube.Execution but got '%T'", obj)
	}
	result := execution.ExecutionResult

	ui.Warn("ID:      ", execution.Id)
	ui.Warn("Name:    ", execution.Name)
	ui.Warn("Args:    ", execution.Args...)
	ui.Warn("Duration:", execution.Duration)

	if result == nil {
		return fmt.Errorf("got execution without `Result`")
	}

	ui.NL()

	switch true {
	case result.IsQueued():
		ui.Warn("Test queued for execution")

	case result.IsPending():
		ui.Warn("Test execution started")

	case result.IsSuccesful():
		ui.Info(result.Output)
		duration := execution.EndTime.Sub(execution.StartTime)
		ui.Success("Test execution completed with sucess in " + duration.String())

	case result.IsFailed():
		ui.Warn("Test test execution failed:\n")
		ui.Errf(result.ErrorMessage)
		ui.Info(result.Output)
		os.Exit(1)
	}

	ui.NL()

	return nil
}
