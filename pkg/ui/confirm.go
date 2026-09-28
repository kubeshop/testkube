package ui

import "github.com/pterm/pterm"

func (ui *UI) Confirm(label string) bool {
	ui.requireInteractive(label)

	ok, err := pterm.DefaultInteractiveConfirm.
		WithDefaultValue(true).
		Show(label)
	ui.ExitOnError("reading confirmation", err)

	ui.NL()

	return ok
}
