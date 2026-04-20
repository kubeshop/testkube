package common

import "github.com/kubeshop/testkube/pkg/ui"

// UIShellGetExecution prints kubectl command to get test workflow execution details.
func UIShellGetExecution(id string) {
	ui.ShellCommand(
		"Use following command to get test workflow execution details",
		"testkube get twe "+id,
	)
}

// UIShellViewExecution prints kubectl command to view test workflow execution in the browser.
func UIShellViewExecution(id string) {
	ui.ShellCommand(
		"View test workflow execution in your browser",
		"testkube view "+id,
	)
}

// UIShellWatchExecution prints kubectl command to watch a test workflow execution until complete.
func UIShellWatchExecution(id string) {
	ui.ShellCommand(
		"Watch test workflow execution until complete",
		"testkube watch twe "+id,
	)
}
