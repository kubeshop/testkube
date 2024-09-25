package bufferedstream

import (
	"context"
	"io"
	"os"

	"github.com/pkg/errors"
)

// bufferedStream is a mechanism to buffer data in FS instead of memory.
type bufferedStream struct {
	ctx  context.Context
	end  context.CancelCauseFunc
	file *os.File
	size int
}

type BufferedStream interface {
	io.Reader
	Len() int
	Cleanup() error
	Err() error
	Ready() <-chan struct{}
}

func newBufferedStream(file *os.File, source io.Reader) BufferedStream {
	ctx, end := context.WithCancelCause(context.Background())
	stream := &bufferedStream{ctx: ctx, end: end, file: file}

	// Stream the data into file buffer
	go func() {
		size, err := io.Copy(file, source)
		stream.size = int(size)
		if err == nil || errors.Is(err, io.EOF) {
			_, err = file.Seek(0, io.SeekStart)
		}
		if err == nil {
			err = io.EOF
		}
		stream.end(err)
	}()

	return stream
}

func (b *bufferedStream) Read(p []byte) (n int, err error) {
	<-b.ctx.Done()
	return b.file.Read(p)
}

func (b *bufferedStream) Err() error {
	return context.Cause(b.ctx)
}

func (b *bufferedStream) Ready() <-chan struct{} {
	return b.ctx.Done()
}

func (b *bufferedStream) Len() int {
	<-b.ctx.Done()
	return b.size
}

func (b *bufferedStream) Cleanup() error {
	b.end(nil)
	return os.Remove(b.file.Name())
}

func NewBufferedStream(dirPath, prefix string, source io.Reader) (BufferedStream, error) {
	file, err := os.CreateTemp(dirPath, prefix)
	if err != nil {
		return nil, err
	}
	return newBufferedStream(file, source), nil
}
