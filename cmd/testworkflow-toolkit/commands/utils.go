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

// Run runs the command. When the command fails, the error also has the diagnostic line of
// the standard error, because the exit code alone does not tell the cause.
func Run(c string, args ...interface{}) error {
	sub := Comm(c, args...)
	diagnostic := &diagnosticLine{}
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
			return fmt.Errorf("%w: %s", err, line)
		}
	}
	return err
}

// maxDiagnosticLineSize limits the memory for one line of the standard error.
const maxDiagnosticLineSize = 4096

// diagnosticLine keeps the last line of the output that starts with "fatal:", or else the
// last line that is not empty. Git writes the cause of a failure in a "fatal:" line and
// can write hints after it.
type diagnosticLine struct {
	current []byte
	last    string
	fatal   string
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
	if strings.HasPrefix(line, "fatal:") {
		d.fatal = line
	}
}

// Line returns the diagnostic line, or an empty string when the output has no text.
func (d *diagnosticLine) Line() string {
	d.endLine()
	if d.fatal != "" {
		return d.fatal
	}
	return d.last
}

func RunWithRetry(retries int, delay time.Duration, c string, args ...interface{}) (err error) {
	for i := 0; i < retries; i++ {
		err = Run(c, args...)
		if err == nil {
			return nil
		}
		if i+1 < retries {
			nextDelay := time.Duration(i+1) * delay
			fmt.Printf("error, trying again in %s (attempt %d/%d): %s\n", nextDelay.String(), i+2, retries, err.Error())
			time.Sleep(nextDelay)
		}
	}
	return err
}
