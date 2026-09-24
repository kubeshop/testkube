package ui

import "github.com/pterm/pterm"

func (ui *UI) TextInput(text string, defaultValue ...string) string {
	ui.requireInteractive(text)

	t := pterm.DefaultInteractiveTextInput.WithMultiLine(false)
	if len(defaultValue) > 0 {
		t = t.WithDefaultValue(defaultValue[0])
	}

	in, err := t.Show(text)
	ui.ExitOnError("reading input", err)

	return in
}
