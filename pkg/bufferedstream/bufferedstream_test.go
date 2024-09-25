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
