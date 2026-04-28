package common

import (
	"github.com/kubeshop/testkube/pkg/ui"
)

// UIShellGetExecution prints the testkube command to get test workflow execution details.
func UIShellGetExecution(id string) {
	ui.Hint("Get the TestWorkflow execution details:")
	ui.ShellCommand(
		"testkube get twe " + id,
	)
}

// UIShellViewExecution prints the testkube command to view a test workflow execution in the browser.
func UIShellViewExecution(id string) {
	ui.Hint("View the TestWorkflow execution details in your browser:")
	ui.ShellCommand(
		"testkube view " + id,
	)
}

// UIShellWatchExecution prints the testkube command to watch a test workflow execution until complete.
func UIShellWatchExecution(id string) {
	ui.Hint("Watch the TestWorkflow execution until complete:")
	ui.ShellCommand(
		"testkube watch twe " + id,
	)
}
