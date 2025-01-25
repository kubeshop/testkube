package utilisation

import (
	"bufio"
	"context"
	"fmt"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/pkg/errors"
	"io"
	"os"
	"time"
)

type Writer interface {
	Write(ctx context.Context, data string) error
	Close() error
}

type STDOUTWriter struct{}

func NewSTDOUTWriter() *STDOUTWriter {
	return &STDOUTWriter{}
}

func (w *STDOUTWriter) Write(ctx context.Context, data string) error {
	fmt.Println(data)
	return nil
}

func (w *STDOUTWriter) Close() error {
	return nil
}

type BufferedFileWriter struct {
	f      *os.File
	writer *bufio.Writer
}

func NewBufferedFileWriter(name string) (*BufferedFileWriter, error) {
	f, err := os.CreateTemp("", fmt.Sprintf("metrics-%s", name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create temp file")
	}
	return &BufferedFileWriter{
		f:      f,
		writer: bufio.NewWriter(f),
	}, nil
}

func (w *BufferedFileWriter) Write(ctx context.Context, data string) error {
	_, err := w.writer.WriteString(data + "\n")
	if err != nil {
		return errors.Wrapf(err, "failed to write to file")
	}
	return nil
}

func (w *BufferedFileWriter) Close() error {
	if err := w.writer.Flush(); err != nil {
		return errors.Wrapf(err, "failed to flush writer")
	}
	if err := w.f.Close(); err != nil {
		return errors.Wrapf(err, "failed to close file")
	}
	return nil
}

func (w *BufferedFileWriter) Print() error {
	fmt.Printf("Opening metrics file %s\n", w.f.Name())
	f, err := os.Open(w.f.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to open file")
	}
	fmt.Println("Printing file content")
	data, err := io.ReadAll(f)
	if err != nil {
		return errors.Wrapf(err, "failed to read file")
	}
	fmt.Println(string(data))
	return nil
}

func WithMetricsRecorder(step string, fn func()) {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := NewBufferedFileWriter(step)
	if err != nil {
		// log the error
		stdoutUnsafe.Error(err.Error())
		// run the function without metrics
		fn()
		// exit early
		return
	}
	// create the metrics recorder
	tags := []KeyValue{
		NewKeyValue("step", step),
	}
	r := NewMetricsRecorder(WithTags(tags), WithWriter(w))
	go func() {
		r.Start(cancelCtx)
	}()
	// run the function
	fn()
	cancel()
	time.Sleep(1 * time.Second)
	if err := w.Print(); err != nil {
		stdoutUnsafe.Error(err.Error())
	}
}
