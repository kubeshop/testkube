package ui

import "github.com/pterm/pterm"

func (ui *UI) TextInput(text string, defaultValue ...string) string {
	t := pterm.DefaultInteractiveTextInput.WithMultiLine(false)
	if len(defaultValue) > 0 {
		t = t.WithDefaultValue(defaultValue[0])
	}

	in, _ := t.Show(text)
	return in
}
