// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package utils

import (
	"bytes"
	"errors"
	"io"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
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
		instruction, _, _ := data.DetectInstruction(lines[i])
		if instruction == nil && i == len(lines)-1 {
			o.lastLine = lines[i]
		}
		if instruction != nil && instruction.Value != nil {
			data.State.SetOutput(instruction.Ref, instruction.Name, instruction.Value)
		}
	}

	// Pass the output down
	return o.writer.Write(p)
}

func (o *outputProcessor) Close() error {
	o.closed = true
	return nil
}
