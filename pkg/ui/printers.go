// simple ui - TODO use something more sophisticated :)
package ui

import (
	"fmt"
	"os"
	"strings"
)

// IconMedal emoji
const IconMedal = "🥇"

// IconError emoji
const IconError = "💔"

func NL() {
	fmt.Println()
}

// Warn shows warning in terminal
func Success(message string, subMessages ...string) {
	fmt.Printf("%s", LightYellow(message))
	for _, sub := range subMessages {
		fmt.Printf(" %s", LightCyan(sub))
	}
	fmt.Printf(" " + IconMedal)
	fmt.Println()
}

// Warn shows warning in terminal
func Warn(message string, subMessages ...string) {
	fmt.Printf("%s", LightYellow(message))
	for _, sub := range subMessages {
		fmt.Printf(" %s", LightCyan(sub))
	}
	fmt.Println()
}

func LogLine(message string) {
	fmt.Printf("%s\n", DarkGray(message))
}

func Info(message string, subMessages ...string) {
	fmt.Printf("%s", DarkGray(message))
	for _, sub := range subMessages {
		fmt.Printf(" %s", LightGray(sub))
	}
	fmt.Println()
}

func Err(err error) {
	fmt.Printf("%s %s %s\n", LightRed("⨯"), Red(err.Error()), IconError)
}

func Errf(err string, params ...interface{}) {
	fmt.Printf("%s %s\n", LightRed("⨯"), Red(fmt.Sprintf(err, params...)))
}

func Fail(err error) {
	Err(err)
	fmt.Println()
	os.Exit(1)
}

func Failf(err string, params ...interface{}) {
	Errf(err, params...)
	fmt.Println()
	os.Exit(1)
}

func CommandOutput(output []byte, command string, params ...string) {
	fullCommand := fmt.Sprintf("%s %s", LightCyan(command), DarkGray(strings.Join(params, " ")))
	fmt.Printf("command: %s\noutput:\n%s\n", LightGray(fullCommand), DarkGray(string(output)))
}

func Medal() {
	Completed("Congratulations! - Here's your medal: " + IconMedal)
}

func Completed(main string, sub ...string) {
	fmt.Println()
	if len(sub) == 1 {
		fmt.Printf("%s: %s\n", LightGray(main), LightBlue(sub[0]))
	} else {
		fmt.Println(LightGray(main), LightBlue(strings.Join(sub, " ")))
	}
}

func GroupCompleted(main string, sub ...string) {
	fmt.Println()
	line := strings.Repeat("=", calculateMessageLength(main, sub...))
	fmt.Println(LightBlue(line))
	if len(sub) == 1 {
		fmt.Printf("%s: %s", LightGray(main), LightBlue(sub[0]))
	} else {
		fmt.Println(LightGray(main))
	}
}

func InfoGrid(table map[string]string) {
	for k, v := range table {
		fmt.Printf("  %s: %s\n", DarkGray(k), LightBlue(v))
	}
	fmt.Println()
}

func Vector(table []string) {
	for _, v := range table {
		fmt.Printf("  %s\n", DarkGray(v))
	}
}

// Warn shows warning in terminal
func ShellCommand(title string, commands ...string) {
	fmt.Printf("%s:\n", White(title))
	for _, sub := range commands {
		fmt.Printf("$ %s\n", LightGray(sub))
	}
	fmt.Println()
}

func calculateMessageLength(message string, subMessages ...string) int {
	sum := 0
	for _, sub := range subMessages {
		sum += len(sub) + 1 // space
	}

	return sum + len(message)
}
