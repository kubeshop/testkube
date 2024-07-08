package data

import (
	"bytes"
	"errors"
	"io"
)

type outputProcessor struct {
	writer   io.Writer
	ref      string
	closed   bool
	lastLine []byte
}

func NewOutputProcessor(ref string, writer io.Writer) io.WriteCloser {
	return &outputProcessor{
		writer: writer,
		ref:    ref,
	}
}

func (o *outputProcessor) Write(p []byte) (int, error) {
	if o.closed {
		return 0, errors.New("stream is already closed")
	}

	// Process to search for output
	lines := bytes.Split(append(o.lastLine, p...), []byte("\n"))
	o.lastLine = nil
	for i := range lines {
		instruction, _, _ := DetectInstruction(lines[i])
		if instruction == nil && i == len(lines)-1 {
			o.lastLine = lines[i]
		}
		if instruction != nil && instruction.Value != nil {
			GetState().SetOutput(instruction.Ref, instruction.Name, instruction.Value)
		}
	}

	// Pass the output down
	return o.writer.Write(p)
}

func (o *outputProcessor) Close() error {
	o.closed = true
	return nil
}
