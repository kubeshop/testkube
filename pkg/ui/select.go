package ui

import "github.com/pterm/pterm"

func (ui *UI) Select(label string, options []string) string {
	ui.requireInteractive(label)

	val, err := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithDefaultText(label).
		Show()
	ui.ExitOnError("reading selection", err)

	ui.NL()

	return val
}
