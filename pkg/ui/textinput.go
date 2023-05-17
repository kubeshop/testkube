package ui

import "github.com/pterm/pterm"

func (ui *UI) TextInput(text string) string {
	in, _ := pterm.DefaultInteractiveTextInput.
		WithMultiLine(false).
		Show(text)

	return in
}
