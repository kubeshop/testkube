package ui

import "github.com/pterm/pterm"

func (ui *UI) Select(label string, options []string) string {
	val, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		Show()

	ui.NL()

	return val
}
