package core

import (
	"context"
	errors2 "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

const (
	metadataControlByte      byte = '#'
	metadataControlByteIndex      = 0
	metadataStartIndex            = 1
	headerEndIndex                = 511
	headerEndByte                 = '\n'
	dataStartIndex                = 512
	headerLength                  = dataStartIndex
)

type Writer interface {
	Write(context.Context, string) error
	writeMetadata(context.Context, *Metadata) error
	Close(ctx context.Context) error
}

type STDOUTWriter struct{}

func NewSTDOUTWriter() *STDOUTWriter {
	return &STDOUTWriter{}
}

func (w *STDOUTWriter) Write(ctx context.Context, data string) error {
	fmt.Println(data)
	return nil
}

func (w *STDOUTWriter) writeMetadata(ctx context.Context, metadata *Metadata) error {
	fmt.Printf("Metadata: %v\n", metadata)
	return nil
}

func (w *STDOUTWriter) Close(ctx context.Context) error {
	return nil
}

var _ Writer = &STDOUTWriter{}

type FileWriter struct {
	mu       sync.Mutex
	stop     bool
	f        *os.File
	metadata *Metadata
	// increment specifies by how much the line count should be incremented,
	// this is configured when batching multiple lines into a single write operation.
	increment int
}

// NewFileWriter creates a new FileWriter that writes to a file in the specified directory with the given metadata.
func NewFileWriter(dir string, metadata *Metadata, increment int) (*FileWriter, error) {
	base := fmt.Sprintf("%s_%s_%s", metadata.Workflow, metadata.Step.Ref, metadata.Execution)
	if metadata.Step.Parent != "" {
		base = fmt.Sprintf("%s_%s", base, metadata.Step.Parent)
	}
	if metadata.Index != "" {
		base = fmt.Sprintf("%s_%s", base, metadata.Index)
	}
	filename := fmt.Sprintf("%s.%s", base, metadata.Format)
	f, err := initFile(dir, filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &FileWriter{
		f:         f,
		metadata:  metadata,
		increment: increment,
	}, nil
}

// initFile creates a new file in the specified directory with the given name, reserves space for metadata,
// and moves the cursor to the start of the data section.
func initFile(dir, name string) (*os.File, error) {
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
	if err := reserveMetadataSpace(f); err != nil {
		_ = os.Remove(filePath)
		return nil, errors.Wrap(err, "failed to reserve metadata space in file")
	}
	if _, err := f.Seek(dataStartIndex, io.SeekStart); err != nil {
		_ = os.Remove(filePath)
		return nil, errors.Wrap(err, "failed to seek to data start in file")
	}

	return f, nil
}

// reserveMetadataSpace writes null bytes to the start of the file to reserve space for metadata and a newline as the last character.
func reserveMetadataSpace(f *os.File) error {
	buffer := make([]byte, headerEndIndex+1)
	buffer[headerEndIndex] = headerEndByte
	if _, err := f.WriteAt(buffer, 0); err != nil {
		return errors.Wrapf(err, "failed to reserve metadata space in file")
	}
	return nil
}

// Write writes the given data to the file and appends a newline character.
func (w *FileWriter) Write(ctx context.Context, data string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stop {
		return errors.New("cannot write to stopped writer")
	}
	_, err := w.f.WriteString(data + "\n")
	if err != nil {
		return errors.Wrapf(err, "failed to write to file")
	}
	w.metadata.Lines += w.increment
	return nil
}

func (w *FileWriter) writeMetadata(ctx context.Context, metadata *Metadata) error {
	if w.stop {
		return errors.New("cannot write metadata to closed writer")
	}
	if _, err := w.f.WriteAt([]byte{metadataControlByte}, metadataControlByteIndex); err != nil {
		return errors.Wrapf(err, "failed to write metadata control byte to file")
	}
	serialized := metadata.String()
	if _, err := w.f.WriteAt([]byte(serialized), metadataStartIndex); err != nil {
		return errors.Wrapf(err, "failed to write metadata to file")
	}

	return nil
}

func (w *FileWriter) Close(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Write metadata to the file.
	wErr := w.writeMetadata(ctx, w.metadata)
	if wErr != nil {
		wErr = errors.Wrap(wErr, "failed to write metadata to the file")
	}
	w.stop = true
	cErr := w.f.Close()
	if cErr != nil {
		cErr = errors.Wrapf(cErr, "failed to close file")
	}
	return errors2.Join(wErr, cErr)
}

func (w *FileWriter) Print() error {
	f, err := os.Open(w.f.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to open file")
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return errors.Wrapf(err, "failed to read file")
	}
	fmt.Println(string(data))
	return nil
}

var _ Writer = &FileWriter{}
