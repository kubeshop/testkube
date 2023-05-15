package ui

import "github.com/pterm/pterm"

func (ui *UI) Confirm(label string) bool {
	ok, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultValue(true).
		Show(label)

	ui.NL()

	return ok
}
