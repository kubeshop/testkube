package core

import (
	"bufio"
	"context"
	errors2 "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
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
	mu       sync.Mutex
	stop     bool
	f        *os.File
	metadata *Metadata
	writer   *bufio.Writer
}

func NewBufferedFileWriter(dir string, metadata *Metadata) (*BufferedFileWriter, error) {
	filename := fmt.Sprintf("%s_%s_%s.%s", metadata.Workflow, metadata.Step, metadata.Execution, metadata.Format)
	f, err := initMetricsFile(dir, filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &BufferedFileWriter{
		f:        f,
		writer:   bufio.NewWriter(f),
		metadata: metadata,
	}, nil
}

func initMetricsFile(dir, name string) (*os.File, error) {
	// Ensure the metrics directory exists, creating it if necessary.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to create %q directory", dir)
	}

	// Create the file inside the metrics directory.
	filePath := filepath.Join(dir, name)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create file: %s", filePath)
	}

	return f, nil
}

func (w *BufferedFileWriter) Write(ctx context.Context, data string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stop {
		return errors.New("cannot write to stopped writer")
	}
	_, err := w.writer.WriteString(data + "\n")
	if err != nil {
		return errors.Wrapf(err, "failed to write to file")
	}
	w.metadata.Lines++
	return nil
}

func (w *BufferedFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stop = true
	// Flush any remaining data to the file.
	if err := w.writer.Flush(); err != nil {
		return errors.Wrapf(err, "failed to flush writer")
	}
	// Write metadata to the file.
	wErr := WriteMetadataToFile(w.f, w.metadata)
	if wErr != nil {
		wErr = errors.Wrap(wErr, "failed to write metadata to the file")
	}
	cErr := w.f.Close()
	if cErr != nil {
		cErr = errors.Wrapf(cErr, "failed to close file")
	}
	return errors2.Join(wErr, cErr)
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
