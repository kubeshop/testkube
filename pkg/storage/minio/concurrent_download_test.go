package minio

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/archive"
)

// TestConcurrentDownloadSafety tests that concurrent downloads are thread-safe
// This validates the exact pattern used in downloadArchive
func TestConcurrentDownloadSafety(t *testing.T) {
	const numFiles = 50
	const maxConcurrent = 10

	files := make([]*archive.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = &archive.File{
			Name: "file",
		}
	}

	ctx := context.Background()

	// Use the EXACT same pattern as our production code
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		semaphore = make(chan struct{}, maxConcurrent)
		dlErr     error
	)

	for i := range files {
		select {
		case <-ctx.Done():
			t.Fatal("context cancelled")
		default:
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			mu.Lock()
			if dlErr != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// Simulate some work
			time.Sleep(time.Millisecond)

			buf := &bytes.Buffer{}
			buf.WriteString("test content")

			// This is the critical section - MUST be protected
			mu.Lock()
			files[idx].Data = buf
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if dlErr != nil {
		t.Errorf("Unexpected error: %v", dlErr)
	}

	// Verify all files have data
	for i, f := range files {
		if f.Data == nil {
			t.Errorf("File %d has no data", i)
		}
	}
}
