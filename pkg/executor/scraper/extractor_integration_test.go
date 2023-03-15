//go:build integration

package scraper_test

import (
	"context"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemExtractor_Extract_Integration(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)

	defer os.RemoveAll(tempDir)

	err = os.Mkdir(filepath.Join(tempDir, "subdir"), os.ModePerm)
	require.NoError(t, err)

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	file3 := filepath.Join(tempDir, "subdir", "file3.txt")

	err = os.WriteFile(file1, []byte("test1"), os.ModePerm)
	assert.NoError(t, err)

	err = os.WriteFile(file2, []byte("test2"), os.ModePerm)
	assert.NoError(t, err)

	err = os.WriteFile(file3, []byte("test3"), os.ModePerm)
	assert.NoError(t, err)

	processCallCount := 0
	processFn := func(ctx context.Context, object *scraper.Object) error {
		processCallCount++
		b, err := io.ReadAll(object.Data)
		if err != nil {
			t.Fatalf("error reading %s: %v", object.Name, err)
		}
		switch object.Name {
		case "file1.txt":

			assert.Equal(t, b, []byte("test1"))
		case "file2.txt":
			assert.Equal(t, b, []byte("test2"))
		case "subdir/file3.txt":
			assert.Equal(t, b, []byte("test3"))
		default:
			t.Fatalf("unexpected file: %s", object.Name)
		}

		return nil
	}

	extractor := scraper.NewFilesystemExtractor([]string{tempDir}, filesystem.NewOSFileSystem())
	err = extractor.Extract(context.Background(), processFn)
	require.NoError(t, err)
	assert.Equal(t, processCallCount, 3)
}
