package scraper_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestArchiveFilesystemExtractor_Extract_NoMeta_Integration(t *testing.T) {
	test.IntegrationTest(t)
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
		assert.Equal(t, "artifacts.tar.gz", object.Name)
		assert.Equal(t, scraper.DataTypeTarball, object.DataType)

		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		if !strings.Contains(path, "file1.txt") && !strings.Contains(path, "file2.txt") && !strings.Contains(path, "subdir/file3.txt") {
			t.Fatalf("Unexpected path: %s", path)
		}
		return nil
	}

	extractor := scraper.NewArchiveFilesystemExtractor(filesystem.NewOSFileSystem())
	scrapeDirs := []string{tempDir}
	masks := []string{".*"}
	err = extractor.Extract(context.Background(), scrapeDirs, masks, processFn, notifyFn)
	require.NoError(t, err)
	assert.Equal(t, 1, processCallCount)
}

func TestArchiveFilesystemExtractor_Extract_Meta_Integration(t *testing.T) {
	test.IntegrationTest(t)
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
		switch object.Name {
		case "artifacts.tar.gz":
			processCallCount++
			assert.Equal(t, scraper.DataTypeTarball, object.DataType)
		case ".testkube-meta-files.json":
			processCallCount++
			var meta scraper.FilesMeta
			jsonData, err := io.ReadAll(object.Data)
			if err != nil {
				t.Fatalf("Failed to read meta files: %v", err)
			}
			if err := json.Unmarshal(jsonData, &meta); err != nil {
				t.Fatalf("Failed to unmarshal meta files: %v", err)
			}
			assert.Len(t, meta.Files, 3)
			assert.Equal(t, "artifacts.tar.gz", meta.Archive)
			assert.Equal(t, scraper.DataTypeTarball, meta.DataType)
			assert.Equal(t, "file1.txt", meta.Files[0].Name)
			assert.Equal(t, int64(5), meta.Files[0].Size)
			assert.Equal(t, "file2.txt", meta.Files[1].Name)
			assert.Equal(t, int64(5), meta.Files[1].Size)
			assert.Equal(t, "subdir/file3.txt", meta.Files[2].Name)
			assert.Equal(t, int64(5), meta.Files[2].Size)
			assert.Equal(t, scraper.DataTypeRaw, object.DataType)
		default:
			t.Fatalf("Unexpected object name: %s", object.Name)
		}

		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		if !strings.Contains(path, "file1.txt") && !strings.Contains(path, "file2.txt") && !strings.Contains(path, "subdir/file3.txt") {
			t.Fatalf("Unexpected path: %s", path)
		}
		return nil
	}

	extractor := scraper.NewArchiveFilesystemExtractor(filesystem.NewOSFileSystem(), scraper.GenerateTarballMetaFile())
	scrapeDirs := []string{tempDir}
	masks := []string{".*"}
	err = extractor.Extract(context.Background(), scrapeDirs, masks, processFn, notifyFn)
	require.NoError(t, err)
	assert.Equal(t, 2, processCallCount)
}

func TestRecursiveFilesystemExtractor_Extract_Integration(t *testing.T) {
	test.IntegrationTest(t)
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
		assert.Equal(t, scraper.DataTypeRaw, object.DataType)
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

	notifyFn := func(ctx context.Context, path string) error {
		if !strings.Contains(path, "file1.txt") && !strings.Contains(path, "file2.txt") && !strings.Contains(path, "subdir/file3.txt") {
			t.Fatalf("unexpected file: %s", path)
		}

		return nil
	}

	extractor := scraper.NewRecursiveFilesystemExtractor(filesystem.NewOSFileSystem())
	scrapeDirs := []string{tempDir, "/nonexistent"}
	masks := []string{".*"}
	err = extractor.Extract(context.Background(), scrapeDirs, masks, processFn, notifyFn)
	require.NoError(t, err)
	assert.Equal(t, processCallCount, 3)
}

func TestRecursiveFilesystemExtractor_Extract_RelPath_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)

	defer os.RemoveAll(tempDir)

	err = os.Mkdir(filepath.Join(tempDir, "subdir"), os.ModePerm)
	require.NoError(t, err)

	file1 := filepath.Join(tempDir, "file1.txt")

	err = os.WriteFile(file1, []byte("test1"), os.ModePerm)
	assert.NoError(t, err)

	processCallCount := 0
	processFn := func(ctx context.Context, object *scraper.Object) error {
		processCallCount++
		b, err := io.ReadAll(object.Data)
		if err != nil {
			t.Fatalf("error reading %s: %v", object.Name, err)
		}
		assert.Equal(t, b, []byte("test1"))
		assert.Equal(t, scraper.DataTypeRaw, object.DataType)
		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		if !strings.Contains(path, "file1.txt") {
			t.Fatalf("unexpected path: %s", path)
		}
		return nil
	}

	extractor := scraper.NewRecursiveFilesystemExtractor(filesystem.NewOSFileSystem())
	scrapeDirs := []string{filepath.Join(tempDir, "file1.txt"), "/nonexistent"}
	masks := []string{".*"}
	err = extractor.Extract(context.Background(), scrapeDirs, masks, processFn, notifyFn)
	require.NoError(t, err)
	assert.Equal(t, processCallCount, 1)
}
