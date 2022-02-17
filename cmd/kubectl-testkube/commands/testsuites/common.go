package testsuites

import (
	"os"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func printTestExecutionDetails(execution testkube.TestSuiteExecution, startTime time.Time) {
	ui.Warn("Id:      ", execution.Id)
	ui.Warn("Name:    ", execution.Name)
	if execution.Status != nil {
		ui.Warn("Status:  ", string(*execution.Status))
	}
	ui.Warn("Duration:", execution.CalculateDuration().String()+"\n")
	ui.Table(execution, os.Stdout)

	ui.NL()
	ui.NL()
}

func uiPrintTestStatus(execution testkube.TestSuiteExecution) {
	switch execution.Status {
	case testkube.TestSuiteExecutionStatusQueued:
		ui.Warn("Test Suite queued for execution")

	case testkube.TestSuiteExecutionStatusPending:
		ui.Warn("Test Suite execution started")

	case testkube.TestSuiteExecutionStatusSuccess:
		ui.Success("Test execution completed with sucess in " + execution.Duration)

	case testkube.TestSuiteExecutionStatusError:
		ui.Errf("Test execution failed")
		os.Exit(1)
	}

	ui.NL()
}

func uiShellTestGetCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube testsuites execution "+id,
	)

	ui.NL()
}

func uiShellTestWatchCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube testsuites watch "+id,
	)

	ui.NL()
}
