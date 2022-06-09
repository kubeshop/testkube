package renderer

import (
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func RenderExecutionResult(result *testkube.ExecutionResult) {

	if result == nil {
		ui.Errf("got execution without `Result`")
		return
	}

	ui.NL()

	switch true {
	case result.IsQueued():
		ui.Warn("Status", "test queued for execution")

	case result.IsRunning():
		ui.Warn("Test execution started")

	case result.IsPassed():
		ui.Info(result.Output)
		ui.Success("Status", "Test execution completed with success")

	case result.IsFailed():
		ui.UseStderr()
		ui.Warn("Status", "test execution failed:\n")
		ui.Errf(result.ErrorMessage)
		ui.Info(result.Output)
		os.Exit(1)

	default:
		ui.UseStderr()
		ui.Warn("Status", "test execution status unknown:\n")
		ui.Errf(result.ErrorMessage)
		ui.Info(result.Output)
		os.Exit(1)
	}

}
