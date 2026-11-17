package bufferedstream

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBufferedStream(t *testing.T) {
	inputStream := bytes.NewBuffer([]byte("test input stream"))
	file, err := os.CreateTemp("", "testbuffer")
	if err != nil {
		t.Error("failed to create temp file")
		return
	}
	stream := newBufferedStream(file, inputStream)
	defer stream.Cleanup()
	select {
	case <-stream.Ready():
	case <-time.After(1 * time.Second):
		t.Error("timed out waiting for stream to be ready")
		return
	}

	result, _ := io.ReadAll(stream)
	assert.Equal(t, []byte("test input stream"), result)
}

func TestBufferedStream_Cleanup(t *testing.T) {
	inputStream := bytes.NewBuffer([]byte("test input stream"))
	file, err := os.CreateTemp("", "testbuffer")
	if err != nil {
		t.Error("failed to create temp file")
		return
	}
	stream := newBufferedStream(file, inputStream)
	select {
	case <-stream.Ready():
	case <-time.After(1 * time.Second):
		stream.Cleanup()
		t.Error("timed out waiting for stream to be ready")
		return
	}

	statBefore, statBeforeErr := os.Stat(file.Name())

	err = stream.Cleanup()
	stat, statErr := os.Stat(file.Name())

	assert.NoError(t, statBeforeErr)
	assert.NotEqual(t, nil, statBefore)
	assert.NoError(t, err)
	assert.Equal(t, nil, stat)
	assert.Error(t, statErr)
}

func TestBufferedStream_Rewind(t *testing.T) {
	payload := []byte("test input stream")
	inputStream := bytes.NewBuffer(append([]byte{}, payload...))
	file, err := os.CreateTemp("", "testbuffer")
	if err != nil {
		t.Error("failed to create temp file")
		return
	}
	stream := newBufferedStream(file, inputStream)
	defer stream.Cleanup()
	select {
	case <-stream.Ready():
	case <-time.After(1 * time.Second):
		t.Error("timed out waiting for stream to be ready")
		return
	}

	first, _ := io.ReadAll(stream)
	assert.Equal(t, payload, first)

	// After a full read the reader is at EOF; Rewind must reset it to the start.
	assert.NoError(t, stream.Rewind())

	// Len still reports the full size, and a second read returns the full payload.
	assert.Equal(t, len(payload), stream.Len())
	second, _ := io.ReadAll(stream)
	assert.Equal(t, payload, second)
}

func TestBufferedStream_Rewind_BeforeFirstRead(t *testing.T) {
	payload := []byte("test input stream")
	inputStream := bytes.NewBuffer(append([]byte{}, payload...))
	file, err := os.CreateTemp("", "testbuffer")
	if err != nil {
		t.Error("failed to create temp file")
		return
	}
	stream := newBufferedStream(file, inputStream)
	defer stream.Cleanup()

	// Rewind racing the fill goroutine must block on readiness, then succeed.
	done := make(chan error, 1)
	go func() { done <- stream.Rewind() }()
	select {
	case rewindErr := <-done:
		assert.NoError(t, rewindErr)
	case <-time.After(1 * time.Second):
		t.Error("Rewind blocked past readiness")
		return
	}

	result, _ := io.ReadAll(stream)
	assert.Equal(t, payload, result)
}

func TestBufferedStream_Rewind_GateResolvesOnError(t *testing.T) {
	// When the fill goroutine ends via an error path (not EOF), the ctx.Done()
	// gate must still resolve so Rewind returns rather than blocking forever
	// while the caller holds a lock.
	file, err := os.CreateTemp("", "testbuffer")
	if err != nil {
		t.Error("failed to create temp file")
		return
	}
	stream := newBufferedStream(file, errReader{err: io.ErrClosedPipe})
	defer stream.Cleanup()

	done := make(chan error, 1)
	go func() { done <- stream.Rewind() }()
	select {
	case <-done:
		// Returned (no deadlock); seek on the temp file succeeds regardless.
	case <-time.After(1 * time.Second):
		t.Error("Rewind blocked forever on the error-exit path")
	}
}

func TestBufferedStream_Rewind_Empty(t *testing.T) {
	file, err := os.CreateTemp("", "testbuffer")
	if err != nil {
		t.Error("failed to create temp file")
		return
	}
	stream := newBufferedStream(file, bytes.NewBuffer(nil))
	defer stream.Cleanup()
	select {
	case <-stream.Ready():
	case <-time.After(1 * time.Second):
		t.Error("timed out waiting for stream to be ready")
		return
	}

	assert.NoError(t, stream.Rewind())
	result, _ := io.ReadAll(stream)
	assert.Empty(t, result)
}

type errReader struct{ err error }

func (e errReader) Read(_ []byte) (int, error) { return 0, e.err }

func TestBufferedStream_Len(t *testing.T) {
	inputStream := bytes.NewBuffer([]byte("test input stream"))
	inputStreamLen := inputStream.Len()
	file, err := os.CreateTemp("", "testbuffer")
	if err != nil {
		t.Error("failed to create temp file")
		return
	}
	stream := newBufferedStream(file, inputStream)
	defer stream.Cleanup()

	size := -1
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		size = stream.Len()
		wg.Done()
	}()

	select {
	case <-stream.Ready():
	case <-time.After(1 * time.Second):
		t.Error("timed out waiting for stream to be ready")
		return
	}

	wg.Wait()
	assert.Equal(t, inputStreamLen, size)
}
