package common

import (
	"os"

	"github.com/mattn/go-isatty"

	"github.com/kubeshop/testkube/pkg/ui"
)

// Gate hints on a TTY so non-interactive stdout (e.g. `-o json | jq`) stays parseable.
// Var (not func) so tests can override.
var hintsEnabled = func() bool {
	fd := os.Stdout.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func shellHint(hint, cmd string) {
	if !hintsEnabled() {
		return
	}
	ui.Hint(hint)
	ui.ShellCommand(cmd)
}

func UIShellGetExecution(id string) {
	shellHint("Get the TestWorkflow execution details:", "testkube get twe "+id)
}

func UIShellViewExecution(id string) {
	shellHint("View the TestWorkflow execution details in your browser:", "testkube view "+id)
}

func UIShellWatchExecution(id string) {
	shellHint("Watch the TestWorkflow execution until complete:", "testkube watch twe "+id)
}
