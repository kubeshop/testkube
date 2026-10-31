package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func concat(args ...interface{}) []string {
	result := make([]string, 0)
	for _, a := range args {
		switch a := a.(type) {
		case string:
			result = append(result, a)
		case int:
			result = append(result, strconv.Itoa(a))
		case []string:
			result = append(result, a...)
		case []interface{}:
			result = append(result, concat(a...)...)
		}
	}
	return result
}

func Comm(cmd string, args ...interface{}) *exec.Cmd {
	return exec.CommandContext(context.Background(), cmd, concat(args...)...)
}

// runWaitDelay limits the wait for the standard error after the command exits.
var runWaitDelay = 10 * time.Second

// Diagnostics choose the lines of the standard error that the error of a failed command keeps.
type Diagnostics struct {
	// Cause matches a line that names the cause. The last match wins. Without a match, the error
	// keeps the last line that is not empty.
	Cause func(line string) bool
	// Note matches a line that the caller reads apart from the cause, for example a refusal that
	// comes before the cause. The last match wins.
	Note func(line string) bool
}

// Run runs the command. When the command fails, the error also has the diagnostic line of
// the standard error, because the exit code alone does not tell the cause.
func Run(diagnostics Diagnostics, c string, args ...interface{}) error {
	sub := Comm(c, args...)
	diagnostic := &diagnosticLine{diagnostics: diagnostics}
	sub.Stdout = os.Stdout
	sub.Stderr = io.MultiWriter(os.Stderr, diagnostic)
	// A child process of the command can keep the standard error open after the command exits.
	// The delay stops Run from waiting for that child process.
	sub.WaitDelay = runWaitDelay
	err := sub.Run()
	// Wait returns ErrWaitDelay only when the command itself passed.
	if errors.Is(err, exec.ErrWaitDelay) {
		return nil
	}
	if err != nil {
		if line := diagnostic.Line(); line != "" {
			return &CommandError{Err: err, Line: line, Note: diagnostic.note}
		}
	}
	return err
}

// CommandError is the failure of a command with the diagnostic line of its standard error. The
// line names the cause, so a caller can show it without the exit status.
type CommandError struct {
	Err  error
	Line string
	// Note is the last line that the Note match of the diagnostics found.
	Note string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.Line)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// maxDiagnosticLineSize limits the memory for one line of the standard error.
const maxDiagnosticLineSize = 4096

// diagnosticLine keeps the last line of the output that names the cause, or else the last line
// that is not empty, and the last line that the note match finds.
type diagnosticLine struct {
	diagnostics Diagnostics
	current     []byte
	last        string
	cause       string
	note        string
}

func (d *diagnosticLine) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' || b == '\r' {
			d.endLine()
			continue
		}
		if len(d.current) < maxDiagnosticLineSize {
			d.current = append(d.current, b)
		}
	}
	return len(p), nil
}

func (d *diagnosticLine) endLine() {
	line := strings.TrimSpace(string(d.current))
	d.current = d.current[:0]
	if line == "" {
		return
	}
	d.last = line
	if d.diagnostics.Cause != nil && d.diagnostics.Cause(line) {
		d.cause = line
	}
	if d.diagnostics.Note != nil && d.diagnostics.Note(line) {
		d.note = line
	}
}

// Line returns the diagnostic line, or an empty string when the output has no text.
func (d *diagnosticLine) Line() string {
	d.endLine()
	if d.cause != "" {
		return d.cause
	}
	return d.last
}

// RunWithRetry runs the command up to the given number of times. It stops early when final reports
// an error that a retry cannot fix.
func RunWithRetry(retries int, delay time.Duration, final func(error) bool, diagnostics Diagnostics, c string, args ...interface{}) (err error) {
	for i := 0; i < retries; i++ {
		err = Run(diagnostics, c, args...)
		if err == nil {
			return nil
		}
		if final != nil && final(err) {
			return err
		}
		if i+1 < retries {
			nextDelay := time.Duration(i+1) * delay
			fmt.Printf("error, trying again in %s (attempt %d/%d): %s\n", nextDelay.String(), i+2, retries, err.Error())
			time.Sleep(nextDelay)
		}
	}
	return err
}
