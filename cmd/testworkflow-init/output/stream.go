package output

import (
	"io"
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/obfuscator"
)

var (
	Std = NewStream(os.Stdout)
)

type obfuscatorLike interface {
	SetSensitiveWords([]string)
	SetSensitiveReplacer(func([]byte) []byte)
}

type stream struct {
	*printer

	direct *stream
}

func NewStream(dst io.Writer) *stream {
	s := &stream{}
	s.printer = &printer{direct: dst}
	s.through = obfuscator.New(dst, obfuscator.FullReplace("*****"), nil)
	s.direct = &stream{printer: &printer{direct: s.printer.direct, through: s.printer.direct}}
	return s
}

func (s *stream) Direct() *stream {
	return s.direct
}

func (s *stream) SetSensitiveWords(words []string) {
	if v, ok := s.through.(obfuscatorLike); ok {
		v.SetSensitiveWords(words)
	}
}

func (s *stream) SetSensitiveReplacer(replacer func(value []byte) []byte) {
	if v, ok := s.through.(obfuscatorLike); ok {
		v.SetSensitiveReplacer(replacer)
	}
}

func (s *stream) Flush() {
	if v, ok := s.through.(flushWriter); ok {
		v.Flush()
	}
}
