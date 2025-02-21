package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
)

var (
	h1        = pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.Bold)).WithTextStyle(pterm.NewStyle(pterm.FgLightMagenta)).WithMargin(0)
	h2        = pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.Bold)).WithTextStyle(pterm.NewStyle(pterm.FgLightGreen)).WithMargin(0)
	paragraph = pterm.DefaultParagraph.WithMaxWidth(100)
)

// H1 prints h1 like header
func (ui *UI) H1(text string) {
	h1.WithWriter(ui.Writer).Println(text)
}

// H1 prints h2 like header
func (ui *UI) H2(text string) {
	h2.WithWriter(ui.Writer).Println(text)
}

func (ui *UI) Paragraph(text string) {
	paragraph.WithWriter(ui.Writer).Println(text)
}

func (ui *UI) NL(amount ...int) {
	if len(amount) > 0 && amount[0] > 0 {
		for i := 0; i < amount[0]-1; i++ {
			fmt.Fprintln(ui.Writer)
		}
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) DebugNL(amount ...int) {
	if !ui.Verbose {
		return
	}
	ui.NL(amount...)
}

// Success shows success in terminal
func (ui *UI) Success(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, "%s", LightYellow(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightCyan(sub))
	}
	fmt.Fprintf(ui.Writer, " "+IconMedal)
	fmt.Fprintln(ui.Writer)
}

// SuccessAndExit shows success in terminal and exit
func (ui *UI) SuccessAndExit(message string, subMessages ...string) {
	ui.Success(message, subMessages...)
	os.Exit(0)
}

// Warn shows warning in terminal
func (ui *UI) Warn(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, "%s", LightYellow(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightCyan(sub))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) Alert(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, "%s", White(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightRed(sub))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) LogLine(message string) {
	fmt.Fprintf(ui.Writer, "%s\n", LightGray(message))
}

func (ui *UI) LogMultiLine(message string) {
	fmt.Fprintf(ui.Writer, "%s\n\n", LightGray(message))
}

func (ui *UI) Debug(message string, subMessages ...string) {
	if !ui.Verbose {
		return
	}
	fmt.Fprintf(ui.Writer, "%s", DarkGray(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightGray(sub))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) Print(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, "%s", White(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", White(sub))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) Printf(format string, data ...any) {
	fmt.Fprintf(ui.Writer, format, data...)
}

func (ui *UI) PrintDot() {
	fmt.Fprint(ui.Writer, ".")
}

// PrintEnabled shows enabled in terminal
func (ui *UI) PrintEnabled(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, IconMedal+"  ")
	fmt.Fprintf(ui.Writer, "%s", White(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", Green(sub))
	}
	fmt.Fprintln(ui.Writer)
}

// PrintDisabled shows insuccess in terminal
func (ui *UI) PrintDisabled(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, IconCross+"  ")
	fmt.Fprintf(ui.Writer, "%s", White(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightMagenta(sub))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) Info(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, "%s", White(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightGray(sub))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) Err(err error) {
	fmt.Fprintf(ui.Writer, "%s %s %s\n", LightRed("⨯"), Red(err.Error()), IconError)
}

func (ui *UI) Errf(err string, params ...interface{}) {
	fmt.Fprintf(ui.Writer, "%s %s\n", LightRed("⨯"), Red(fmt.Sprintf(err, params...)))
}

func (ui *UI) Fail(err error) {
	ui.Writer = os.Stderr
	ui.Err(err)
	fmt.Fprintln(ui.Writer)
	os.Exit(1)
}

func (ui *UI) Failf(err string, params ...interface{}) {
	ui.Writer = os.Stderr
	ui.Errf(err, params...)
	fmt.Fprintln(ui.Writer)
	os.Exit(1)
}

func (ui *UI) CommandOutput(output []byte, command string, params ...string) {
	fullCommand := fmt.Sprintf("%s %s", White(command), LightGray(strings.Join(params, " ")))
	fmt.Fprintf(ui.Writer, "command: %s\noutput:\n%s\n", White(fullCommand), LightGray(string(output)))
}

func (ui *UI) Medal() {
	ui.Completed("Congratulations! - Here's your medal: " + IconMedal)
}

func (ui *UI) Completed(main string, sub ...string) {
	fmt.Fprintln(ui.Writer)
	if len(sub) == 1 {
		fmt.Fprintf(ui.Writer, "%s: %s\n", LightGray(main), LightBlue(sub[0]))
	} else {
		fmt.Fprintln(ui.Writer, LightGray(main), LightBlue(strings.Join(sub, " ")))
	}
}

func (ui *UI) GroupCompleted(main string, sub ...string) {
	fmt.Fprintln(ui.Writer)
	line := strings.Repeat("=", ui.calculateMessageLength(main, sub...))
	fmt.Fprintln(ui.Writer, LightBlue(line))
	if len(sub) == 1 {
		fmt.Fprintf(ui.Writer, "%s: %s", LightGray(main), LightBlue(sub[0]))
	} else {
		fmt.Fprintln(ui.Writer, LightGray(main))
	}
}

func (ui *UI) InfoGrid(table map[string]string) {
	for k, v := range table {
		fmt.Fprintf(ui.Writer, "  %s: %s\n", LightGray(k), LightBlue(v))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) Properties(table [][]string) {
	for _, properties := range table {
		if len(properties) > 1 && properties[0] == Separator {
			fmt.Fprintln(ui.Writer)
			continue
		}

		if len(properties) == 1 {
			fmt.Fprintf(ui.Writer, "  %s\n", Default(properties[0]))
			fmt.Fprintf(ui.Writer, "  %s\n", Default(strings.Repeat("-", len(properties[0]))))
		}

		if len(properties) == 2 {
			fmt.Fprintf(ui.Writer, "  %s: %s\n", White(properties[0]), LightBlue(properties[1]))
		}
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) Vector(table []string) {
	for _, v := range table {
		fmt.Fprintf(ui.Writer, "  %s\n", LightGray(v))
	}
}

// Warn shows warning in terminal
func (ui *UI) ShellCommand(title string, commands ...string) {
	fmt.Fprintf(ui.Writer, "$ %s", LightGray(title))
	for _, sub := range commands {
		fmt.Fprintf(ui.Writer, LightGray(" \\\n\t%s"), sub)
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) calculateMessageLength(message string, subMessages ...string) int {
	sum := 0
	for _, sub := range subMessages {
		sum += len(sub) + 1 // space
	}

	return sum + len(message)
}

func (ui *UI) Link(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, "%s", LightGray(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightGray(sub))
	}
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) ExecutionLink(message string, subMessages ...string) {
	fmt.Fprintf(ui.Writer, "%s", LightBlue(message))
	for _, sub := range subMessages {
		fmt.Fprintf(ui.Writer, " %s", LightBlue(sub))
	}
	fmt.Fprintln(ui.Writer)
}
