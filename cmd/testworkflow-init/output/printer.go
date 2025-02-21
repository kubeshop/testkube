package output

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/gookit/color"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
)

type FlushWriter interface {
	io.Writer
	Flush()
}

type printer struct {
	through io.Writer
	direct  io.Writer
}

// Write sends bytes, sanitizing it
func (s *printer) Write(p []byte) (n int, err error) {
	n, err = s.through.Write(p)
	if err != nil {
		return n, err
	}
	// On success, the stream needs to return same number of bytes if used as command pipe,
	// otherwise, the child process will receive SIGPIPE.
	return len(p), err
}

// Printf sends a formatted string via stream, sanitizing it
func (s *printer) Printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(s.through, format, args...)
}

// Print sends a bare string via stream, sanitizing it
func (s *printer) Print(message string) {
	_, _ = s.through.Write(unsafe.Slice(unsafe.StringData(message), len(message)))
}

// Println sends a bare string via stream, sanitizing it
func (s *printer) Println(message string) {
	buf := make([]byte, len(message)+1)
	copy(buf, unsafe.Slice(unsafe.StringData(message), len(message)))
	buf[len(buf)-1] = '\n'
	_, _ = s.through.Write(buf)
}

func (s *printer) printfColor(color color.Color, format string, args ...interface{}) {
	s.Printf(color.Render(format), args...)
}

func (s *printer) printColor(color color.Color, message string) {
	s.printfColor(color, "%s", message)
}

// Errorf sends a formatted string via stream, sanitizing it
func (s *printer) Errorf(format string, args ...interface{}) {
	s.printfColor(color.FgRed, format, args...)
}

// Error sends a bare string via stream, sanitizing it
func (s *printer) Error(message string) {
	s.printColor(color.FgRed, message)
}

// Warnf sends a formatted string via stream, sanitizing it
func (s *printer) Warnf(format string, args ...interface{}) {
	s.printfColor(color.FgYellow, format, args...)
}

// Warn sends a bare string via stream, sanitizing it
func (s *printer) Warn(message string) {
	s.printColor(color.FgYellow, message)
}

// Hint sends a hint via stream, bypassing the output sanitization
func (s *printer) Hint(ref, name string) {
	hint := instructions.SprintHint(ref, name)
	_, _ = s.direct.Write(unsafe.Slice(unsafe.StringData(hint), len(hint)))
}

// HintDetails sends a hint via stream, bypassing the output sanitization
func (s *printer) HintDetails(ref, name string, value interface{}) {
	hint := instructions.SprintHintDetails(ref, name, value)
	_, _ = s.direct.Write(unsafe.Slice(unsafe.StringData(hint), len(hint)))
}

// Output sends an instruction via stream, bypassing the output sanitization
func (s *printer) Output(ref, name string, value interface{}) {
	output := instructions.SprintOutput(ref, name, value)
	_, _ = s.direct.Write(unsafe.Slice(unsafe.StringData(output), len(output)))
}
