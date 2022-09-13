package ui

import (
	"fmt"
	"os"
)

func (ui *UI) ExitOnError(item string, errors ...error) {
	ui.printAndExit(item, true, errors...)
}

func (ui *UI) PrintOnError(item string, errors ...error) {
	ui.printAndExit(item, false, errors...)
}

func (ui *UI) printAndExit(item string, exitOnError bool, errors ...error) {
	if len(errors) > 0 && ui.hasErrors(errors...) {
		ui.NL()
		for _, err := range errors {
			if err != nil {
				writer := Writer
				if exitOnError {
					writer = os.Stderr
				}

				fmt.Fprintf(writer, "%s (error: %s)\n\n", Red(item), err)
				if exitOnError {
					os.Exit(1)
				}
			}
		}
	}

	if ui.Verbose {
		fmt.Fprintf(Writer, "%s %s\n", Blue("\xE2\x9C\x94"), Green(item))
	}
}

func (ui *UI) WarnOnError(item string, errors ...error) {
	if len(errors) > 0 && ui.hasErrors(errors...) {
		for _, err := range errors {
			if err != nil {
				fmt.Fprintf(Writer, "%s %s (error: %s)\n\n", LightYellow("⨯"), Yellow(item), err)
				return
			}
		}
	}

	if ui.Verbose {
		fmt.Fprintf(Writer, "%s %s\n", Blue("\xE2\x9C\x94"), Green(item))
	}
}

func (ui *UI) hasErrors(errors ...error) bool {
	if len(errors) > 0 {
		for _, err := range errors {
			if err != nil {
				return true
			}
		}
	}

	return false
}
