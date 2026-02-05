package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/pkg/archive"
)

// mockDownloader simulates a download operation with configurable latency
type mockDownloader struct {
	latency time.Duration
	data    []byte
}

func (m *mockDownloader) download(ctx context.Context) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(m.latency):
		return io.NopCloser(bytes.NewReader(m.data)), nil
	}
}

// downloadSequential simulates the OLD sequential download pattern
func downloadSequential(ctx context.Context, files []*archive.File, downloader *mockDownloader) error {
	for i := range files {
		reader, err := downloader.download(ctx)
		if err != nil {
			return err
		}

		files[i].Data = &bytes.Buffer{}
		if _, err = files[i].Data.ReadFrom(reader); err != nil {
			return err
		}
	}
	return nil
}

// downloadParallelErrgroup simulates the NEW parallel download pattern using errgroup
func downloadParallelErrgroup(ctx context.Context, files []*archive.File, downloader *mockDownloader, maxConcurrent int) error {
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for i := range files {
		idx := i
		g.Go(func() error {
			reader, err := downloader.download(ctx)
			if err != nil {
				return err
			}

			buf := &bytes.Buffer{}
			if _, err = buf.ReadFrom(reader); err != nil {
				return err
			}

			mu.Lock()
			files[idx].Data = buf
			mu.Unlock()

			return nil
		})
	}

	return g.Wait()
}

// BenchmarkDownloadSequential benchmarks the old sequential approach
func BenchmarkDownloadSequential(b *testing.B) {
	downloader := &mockDownloader{
		latency: 10 * time.Millisecond, // Simulate network latency
		data:    bytes.Repeat([]byte("test data "), 1000),
	}

	for _, numFiles := range []int{5, 10, 20, 50} {
		b.Run(fmt.Sprintf("files=%d", numFiles), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				files := make([]*archive.File, numFiles)
				for j := 0; j < numFiles; j++ {
					files[j] = &archive.File{Name: fmt.Sprintf("file%d", j)}
				}

				ctx := context.Background()
				if err := downloadSequential(ctx, files, downloader); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDownloadParallel benchmarks the new parallel approach
func BenchmarkDownloadParallel(b *testing.B) {
	downloader := &mockDownloader{
		latency: 10 * time.Millisecond, // Simulate network latency
		data:    bytes.Repeat([]byte("test data "), 1000),
	}

	for _, numFiles := range []int{5, 10, 20, 50} {
		b.Run(fmt.Sprintf("files=%d", numFiles), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				files := make([]*archive.File, numFiles)
				for j := 0; j < numFiles; j++ {
					files[j] = &archive.File{Name: fmt.Sprintf("file%d", j)}
				}

				ctx := context.Background()
				if err := downloadParallelErrgroup(ctx, files, downloader, 10); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestDownloadPerformanceComparison runs a quick comparison test to show the speedup
func TestDownloadPerformanceComparison(t *testing.T) {
	downloader := &mockDownloader{
		latency: 10 * time.Millisecond,
		data:    bytes.Repeat([]byte("test data "), 100),
	}

	numFiles := 20

	// Sequential timing
	files := make([]*archive.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = &archive.File{Name: fmt.Sprintf("file%d", i)}
	}

	start := time.Now()
	if err := downloadSequential(context.Background(), files, downloader); err != nil {
		t.Fatal(err)
	}
	sequentialDuration := time.Since(start)

	// Parallel timing
	files = make([]*archive.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = &archive.File{Name: fmt.Sprintf("file%d", i)}
	}

	start = time.Now()
	if err := downloadParallelErrgroup(context.Background(), files, downloader, 10); err != nil {
		t.Fatal(err)
	}
	parallelDuration := time.Since(start)

	speedup := float64(sequentialDuration) / float64(parallelDuration)

	t.Logf("Sequential download of %d files: %v", numFiles, sequentialDuration)
	t.Logf("Parallel download of %d files:   %v", numFiles, parallelDuration)
	t.Logf("Speedup: %.2fx faster", speedup)

	// With 20 files, 10ms latency each, and max 10 concurrent:
	// Sequential: ~200ms (20 * 10ms)
	// Parallel: ~20-30ms (2 batches of 10)
	// We expect at least 3x speedup
	if speedup < 3.0 {
		t.Errorf("Expected at least 3x speedup, got %.2fx", speedup)
	}
}
