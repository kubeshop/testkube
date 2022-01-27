package tests

import (
	"os"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func printTestExecutionDetails(execution testkube.TestExecution, startTime time.Time) {
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

func uiPrintTestStatus(execution testkube.TestExecution) {
	switch execution.Status {
	case testkube.TestStatusQueued:
		ui.Warn("Test queued for execution")

	case testkube.TestStatusPending:
		ui.Warn("Test execution started")

	case testkube.TestStatusSuccess:
		ui.Success("Test execution completed with sucess in " + execution.Duration)

	case testkube.TestStatusError:
		ui.Errf("Test execution failed")
		os.Exit(1)
	}

	ui.NL()
}

func uiShellTestGetCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube tests execution "+id,
	)

	ui.NL()
}

func uiShellTestWatchCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube tests watch "+id,
	)

	ui.NL()
}
