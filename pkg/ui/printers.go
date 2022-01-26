// simple ui - TODO use something more sophisticated :)
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var Writer io.Writer = os.Stdout

// IconMedal emoji
const IconMedal = "🥇"

// IconError emoji
const IconError = "💔"

func NL() {
	fmt.Fprintln(Writer)
}

// Warn shows warning in terminal
func Success(message string, subMessages ...string) {
	fmt.Fprintf(Writer, "%s", LightYellow(message))
	for _, sub := range subMessages {
		fmt.Fprintf(Writer, " %s", LightCyan(sub))
	}
	fmt.Fprintf(Writer, " "+IconMedal)
	fmt.Fprintln(Writer)
}

// Warn shows warning in terminal
func Warn(message string, subMessages ...string) {
	fmt.Fprintf(Writer, "%s", LightYellow(message))
	for _, sub := range subMessages {
		fmt.Fprintf(Writer, " %s", LightCyan(sub))
	}
	fmt.Fprintln(Writer)
}

func LogLine(message string) {
	fmt.Fprintf(Writer, "%s\n", DarkGray(message))
}

func Debug(message string, subMessages ...string) {
	if !Verbose {
		return
	}
	fmt.Fprintf(Writer, "%s", DarkGray(message))
	for _, sub := range subMessages {
		fmt.Fprintf(Writer, " %s", LightGray(sub))
	}
	fmt.Fprintln(Writer)
}

func Info(message string, subMessages ...string) {
	fmt.Fprintf(Writer, "%s", DarkGray(message))
	for _, sub := range subMessages {
		fmt.Fprintf(Writer, " %s", LightGray(sub))
	}
	fmt.Fprintln(Writer)
}

func Err(err error) {
	fmt.Fprintf(Writer, "%s %s %s\n", LightRed("⨯"), Red(err.Error()), IconError)
}

func Errf(err string, params ...interface{}) {
	fmt.Fprintf(Writer, "%s %s\n", LightRed("⨯"), Red(fmt.Sprintf(err, params...)))
}

func Fail(err error) {
	Err(err)
	fmt.Fprintln(Writer)
	os.Exit(1)
}

func Failf(err string, params ...interface{}) {
	Errf(err, params...)
	fmt.Fprintln(Writer)
	os.Exit(1)
}

func CommandOutput(output []byte, command string, params ...string) {
	fullCommand := fmt.Sprintf("%s %s", LightCyan(command), DarkGray(strings.Join(params, " ")))
	fmt.Fprintf(Writer, "command: %s\noutput:\n%s\n", LightGray(fullCommand), DarkGray(string(output)))
}

func Medal() {
	Completed("Congratulations! - Here's your medal: " + IconMedal)
}

func Completed(main string, sub ...string) {
	fmt.Fprintln(Writer)
	if len(sub) == 1 {
		fmt.Fprintf(Writer, "%s: %s\n", LightGray(main), LightBlue(sub[0]))
	} else {
		fmt.Fprintln(Writer, LightGray(main), LightBlue(strings.Join(sub, " ")))
	}
}

func GroupCompleted(main string, sub ...string) {
	fmt.Fprintln(Writer)
	line := strings.Repeat("=", calculateMessageLength(main, sub...))
	fmt.Fprintln(Writer, LightBlue(line))
	if len(sub) == 1 {
		fmt.Fprintf(Writer, "%s: %s", LightGray(main), LightBlue(sub[0]))
	} else {
		fmt.Fprintln(Writer, LightGray(main))
	}
}

func InfoGrid(table map[string]string) {
	for k, v := range table {
		fmt.Fprintf(Writer, "  %s: %s\n", DarkGray(k), LightBlue(v))
	}
	fmt.Fprintln(Writer)
}

func Vector(table []string) {
	for _, v := range table {
		fmt.Fprintf(Writer, "  %s\n", DarkGray(v))
	}
}

// Warn shows warning in terminal
func ShellCommand(title string, commands ...string) {
	fmt.Fprintf(Writer, "%s:\n", White(title))
	for _, sub := range commands {
		fmt.Fprintf(Writer, "$ %s\n", LightGray(sub))
	}
	fmt.Fprintln(Writer)
}

func calculateMessageLength(message string, subMessages ...string) int {
	sum := 0
	for _, sub := range subMessages {
		sum += len(sub) + 1 // space
	}

	return sum + len(message)
}
