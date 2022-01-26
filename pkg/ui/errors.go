package ui

import (
	"fmt"
	"os"
)

func ExitOnError(item string, errors ...error) {
	printAndExit(item, true, errors...)
}

func PrintOnError(item string, errors ...error) {
	printAndExit(item, false, errors...)
}

func printAndExit(item string, exitOnError bool, errors ...error) {
	if len(errors) > 0 && hasErrors(errors...) {
		for _, err := range errors {
			if err != nil {
				fmt.Fprintf(Writer, "%s %s (error: %s)\n\n", LightRed("⨯"), Red(item), err)
				if exitOnError {
					os.Exit(1)
				}
			}
		}
	}

	if Verbose {
		fmt.Fprintf(Writer, "%s %s\n", Blue("\xE2\x9C\x94"), Green(item))
	}
}

func WarnOnError(item string, errors ...error) {
	if len(errors) > 0 && hasErrors(errors...) {
		for _, err := range errors {
			if err != nil {
				fmt.Fprintf(Writer, "%s %s (error: %s)\n\n", LightYellow("⨯"), Yellow(item), err)
				return
			}
		}
	}

	if Verbose {
		fmt.Fprintf(Writer, "%s %s\n", Blue("\xE2\x9C\x94"), Green(item))
	}
}

func hasErrors(errors ...error) bool {
	if len(errors) > 0 {
		for _, err := range errors {
			if err != nil {
				return true
			}
		}
	}

	return false
}
