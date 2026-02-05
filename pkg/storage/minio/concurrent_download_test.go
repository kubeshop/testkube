package minio

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/pkg/archive"
)

// TestConcurrentDownloadSafety tests that concurrent downloads are thread-safe
// This validates the errgroup pattern used in downloadArchive
func TestConcurrentDownloadSafety(t *testing.T) {
	const numFiles = 50
	const maxConcurrent = 10

	files := make([]*archive.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = &archive.File{
			Name: "file",
		}
	}

	var mu sync.Mutex

	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(maxConcurrent)

	for i := range files {
		idx := i
		g.Go(func() error {
			// Simulate some work
			time.Sleep(time.Millisecond)

			buf := &bytes.Buffer{}
			buf.WriteString("test content")

			mu.Lock()
			files[idx].Data = buf
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify all files have data
	for i, f := range files {
		if f.Data == nil {
			t.Errorf("File %d has no data", i)
		}
	}
}

// TestConcurrentDownloadErrorPropagation tests that errors are properly propagated
func TestConcurrentDownloadErrorPropagation(t *testing.T) {
	const numFiles = 20
	const maxConcurrent = 10
	const errorAtIndex = 5

	expectedErr := errors.New("simulated download error")

	files := make([]*archive.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = &archive.File{Name: "file"}
	}

	var mu sync.Mutex

	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(maxConcurrent)

	for i := range files {
		idx := i
		g.Go(func() error {
			time.Sleep(time.Millisecond)

			// Simulate an error at a specific index
			if idx == errorAtIndex {
				return expectedErr
			}

			buf := &bytes.Buffer{}
			buf.WriteString("test content")

			mu.Lock()
			files[idx].Data = buf
			mu.Unlock()

			return nil
		})
	}

	err := g.Wait()
	if err == nil {
		t.Error("Expected an error but got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

// TestConcurrentDownloadContextCancellation tests that context cancellation stops downloads
func TestConcurrentDownloadContextCancellation(t *testing.T) {
	const numFiles = 100
	const maxConcurrent = 10

	files := make([]*archive.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = &archive.File{Name: "file"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var completedCount atomic.Int32

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for i := range files {
		idx := i
		g.Go(func() error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-time.After(10 * time.Millisecond):
			}

			// Cancel after first batch completes
			if completedCount.Add(1) == maxConcurrent {
				cancel()
			}

			buf := &bytes.Buffer{}
			buf.WriteString("test content")

			mu.Lock()
			files[idx].Data = buf
			mu.Unlock()

			return nil
		})
	}

	err := g.Wait()
	if err == nil {
		t.Error("Expected context cancellation error")
	}

	// Not all files should be downloaded due to cancellation
	completed := completedCount.Load()
	t.Logf("Completed %d out of %d files before cancellation", completed, numFiles)

	if completed >= int32(numFiles) {
		t.Error("Expected cancellation to stop some downloads")
	}
}

// TestConcurrentDownloadConcurrencyLimit tests that the concurrency limit is respected
func TestConcurrentDownloadConcurrencyLimit(t *testing.T) {
	const numFiles = 50
	const maxConcurrent = 5

	files := make([]*archive.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = &archive.File{Name: "file"}
	}

	var (
		mu             sync.Mutex
		currentCount   int32
		maxObserved    int32
		currentCountMu sync.Mutex
	)

	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(maxConcurrent)

	for i := range files {
		idx := i
		g.Go(func() error {
			// Track concurrent goroutines
			currentCountMu.Lock()
			currentCount++
			if currentCount > maxObserved {
				maxObserved = currentCount
			}
			currentCountMu.Unlock()

			time.Sleep(5 * time.Millisecond)

			currentCountMu.Lock()
			currentCount--
			currentCountMu.Unlock()

			buf := &bytes.Buffer{}
			buf.WriteString("test content")

			mu.Lock()
			files[idx].Data = buf
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	t.Logf("Max concurrent goroutines observed: %d (limit: %d)", maxObserved, maxConcurrent)

	if maxObserved > int32(maxConcurrent) {
		t.Errorf("Concurrency limit exceeded: observed %d, limit %d", maxObserved, maxConcurrent)
	}
}
