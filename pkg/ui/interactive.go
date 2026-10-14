package ui

import (
	"os"

	"github.com/mattn/go-isatty"
)

// stdinIsInteractive reports whether a prompt can be answered. It is a variable so tests can
// take the non-interactive branch without a pseudo terminal.
var stdinIsInteractive = func() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// StdinIsInteractive reports whether a prompt can be answered, for callers that want to
// choose a different path instead of failing on the prompt.
func StdinIsInteractive() bool { return stdinIsInteractive() }

// requireInteractive exits with an actionable message when there is nobody to answer a
// prompt.
//
// Without it the prompt is drawn and then waits forever: the keyboard library cannot open a
// non-terminal stdin, reports the failure as an empty keypress with no error, and pterm has
// no case for an empty keypress, so the listener spins. Every read allocates, so the garbage
// collector runs flat out too and the process burns more than a core until it is killed.
func (ui *UI) requireInteractive(prompt string) {
	if stdinIsInteractive() {
		return
	}
	ui.Failf("this command stopped to ask %q, but stdin is not a terminal so the question cannot be answered.\n"+
		"  Run it from a terminal, or supply the value as a command flag and it will not be asked.", prompt)
}
