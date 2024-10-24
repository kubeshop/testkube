// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/pterm/pterm"
)

const (
	printItemNameLen = 20
)

var (
	DefaultSpinner = buildDefaultSpinner()
)

func buildDefaultSpinner() pterm.SpinnerPrinter {
	spinner := *pterm.DefaultSpinner.WithSequence(" ◐ ", " ◓ ", " ◑ ", " ◒ ")
	spinner.SuccessPrinter = &pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.SuccessMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.SuccessPrefixStyle,
			Text:  "✓",
		},
	}
	spinner.FailPrinter = &pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.ErrorMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.ErrorPrefixStyle,
			Text:  "×",
		},
	}
	return spinner
}

func PrintHeader(content string) {
	fmt.Println("\n" + color.Blue.Render(color.Bold.Render(content)))
}

func PrintActionHeader(content string) {
	fmt.Println("\n" + color.Magenta.Render(color.Bold.Render(content)))
}

func PrintItem(name, value, hint string) {
	whitespace := strings.Repeat(" ", printItemNameLen-len(name))
	if hint != "" {
		fmt.Printf("%s%s %s %s\n", whitespace, color.Bold.Render(name+":"), value, color.FgDarkGray.Render("("+hint+")"))
	} else {
		fmt.Printf("%s%s %s\n", whitespace, color.Bold.Render(name+":"), value)
	}
}

func PrintSpinner(nameOrLabel ...string) func(name string, err ...error) {
	multi := pterm.DefaultMultiPrinter.WithUpdateDelay(10 * time.Millisecond)
	messages := make(map[string]string, len(nameOrLabel)/2)
	spinners := make(map[string]*pterm.SpinnerPrinter, len(nameOrLabel)/2)

	for i := 0; i < len(nameOrLabel); i += 2 {
		name := nameOrLabel[i]
		messages[name] = nameOrLabel[i+1]
		spinners[name], _ = DefaultSpinner.WithWriter(multi.NewWriter()).Start(messages[name])
	}

	multi.Start()

	return func(name string, errs ...error) {
		if spinners[name] == nil || !spinners[name].IsActive {
			return
		}
		err := errors.Join(errs...)
		if err == nil {
			spinners[name].Success()
		} else {
			spinners[name].Fail(fmt.Sprintf("%s: %s", messages[name], err.Error()))
		}
		time.Sleep(10 * time.Millisecond)
		for _, spinner := range spinners {
			if spinner.IsActive {
				return
			}
		}
		multi.Stop()
	}
}
