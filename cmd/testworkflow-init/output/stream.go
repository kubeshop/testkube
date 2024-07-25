package output

import (
	"io"
	"os"
)

var (
	Std = NewStream(os.Stdout)
)

type stream struct {
	*printer

	direct *stream
}

func NewStream(dst io.Writer) *stream {
	s := &stream{}
	s.printer = &printer{direct: dst}
	s.printer.through = newObfuscator(dst, "*****", nil)
	s.direct = &stream{printer: &printer{direct: s.printer.direct, through: s.printer.direct}}
	return s
}

func (s *stream) Direct() *stream {
	return s.direct
}

func (s *stream) SetSensitiveWords(words []string) {
	if v, ok := s.printer.through.(*obfuscator); ok {
		v.SetSensitiveWords(words)
	}
}

func (s *stream) SetSensitiveReplacement(replacement string) {
	if v, ok := s.printer.through.(*obfuscator); ok {
		v.SetSensitiveReplacement(replacement)
	}
}

func (s *stream) Flush() {
	if v, ok := s.printer.through.(FlushWriter); ok {
		v.Flush()
	}
}
