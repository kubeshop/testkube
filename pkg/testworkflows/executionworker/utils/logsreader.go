package utils

import (
	"io"
	"sync/atomic"
)

type logsReader struct {
	io.WriteCloser
	io.Reader
	finished atomic.Bool
	err      atomic.Value
}

func NewLogsReader() LogsReadWriter {
	reader, writer := io.Pipe()
	return &logsReader{
		Reader:      reader,
		WriteCloser: writer,
	}
}

func (n *logsReader) End(err error) {
	if n.finished.CompareAndSwap(false, true) {
		if err != nil {
			n.err.Store(err)
		}
		n.Close()
	}
}

func (n *logsReader) Err() error {
	err := n.err.Load()
	if err == nil {
		return nil
	}
	return err.(error)
}

type LogsReader interface {
	io.Reader
	Err() error
}

type LogsReadWriter interface {
	LogsReader
	io.WriteCloser
	End(err error)
	Err() error
}
