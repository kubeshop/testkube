package scraper_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/filesystem"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/executor/scraper"
)

func TestRecursiveFilesystemExtractor_Extract(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, nil)
	fs.EXPECT().OpenFileBuffered("/my/directory/file1").Return(bufio.NewReader(strings.NewReader("test")), nil)
	extractor := scraper.NewRecursiveFilesystemExtractor(fs)

	// Set up the expected calls to the mocked fs object
	fs.EXPECT().Walk("/my/directory", gomock.Any()).Return(nil).DoAndReturn(func(_ string, walkFn filepath.WalkFunc) error {
		fileInfo := filesystem.MockFileInfo{
			FName:  "file1",
			FIsDir: false,
		}
		return walkFn("/my/directory/file1", &fileInfo, nil)
	})

	processFn := func(ctx context.Context, object *scraper.Object) error {
		assert.Equal(t, "file1", object.Name)
		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		assert.Equal(t, "/my/directory/file1", path)
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, []string{".*"}, processFn, notifyFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
}

func TestArchiveFilesystemExtractor_Extract_NoMeta(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, nil)
	testContent := "test"
	fs.EXPECT().OpenFileBuffered("/my/directory/file1").Return(bufio.NewReader(strings.NewReader(testContent)), nil)
	testFileInfo := filesystem.MockFileInfo{
		FName:    "/my/directory/file1",
		FSize:    int64(len(testContent)),
		FMode:    0755,
		FModTime: time.Time{},
		FIsDir:   false,
	}
	fs.EXPECT().Stat("/my/directory/file1").Return(&testFileInfo, nil)
	extractor := scraper.NewArchiveFilesystemExtractor(fs)

	// Set up the expected calls to the mocked fs object
	fs.EXPECT().Walk("/my/directory", gomock.Any()).Return(nil).DoAndReturn(func(_ string, walkFn filepath.WalkFunc) error {
		fileInfo := filesystem.MockFileInfo{
			FName:  "file1",
			FIsDir: false,
		}
		return walkFn("/my/directory/file1", &fileInfo, nil)
	})

	processFnCallCount := 0
	processFn := func(ctx context.Context, object *scraper.Object) error {
		processFnCallCount++
		switch object.Name {
		case "artifacts.tar.gz":
			assert.Equal(t, scraper.DataTypeTarball, object.DataType)
		default:
			t.Fatalf("Unexpected object name: %s", object.Name)
		}

		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		assert.Equal(t, "/my/directory/file1", path)
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, []string{".*"}, processFn, notifyFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
	assert.Equal(t, 1, processFnCallCount)
}

func TestArchiveFilesystemExtractor_Extract_Meta(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, nil)
	testContent := "test"
	fs.EXPECT().OpenFileBuffered("/my/directory/file1").Return(bufio.NewReader(strings.NewReader(testContent)), nil)
	testFileInfo := filesystem.MockFileInfo{
		FName:    "/my/directory/file1",
		FSize:    int64(len(testContent)),
		FMode:    0755,
		FModTime: time.Time{},
		FIsDir:   false,
	}
	fs.EXPECT().Stat("/my/directory/file1").Return(&testFileInfo, nil)
	extractor := scraper.NewArchiveFilesystemExtractor(fs, scraper.GenerateTarballMetaFile())

	// Set up the expected calls to the mocked fs object
	fs.EXPECT().Walk("/my/directory", gomock.Any()).Return(nil).DoAndReturn(func(_ string, walkFn filepath.WalkFunc) error {
		fileInfo := filesystem.MockFileInfo{
			FName:  "file1",
			FIsDir: false,
		}
		return walkFn("/my/directory/file1", &fileInfo, nil)
	})

	processFnCallCount := 0
	processFn := func(ctx context.Context, object *scraper.Object) error {
		processFnCallCount++
		switch object.Name {
		case ".testkube-meta-files.json":
			var meta scraper.FilesMeta
			jsonData, err := io.ReadAll(object.Data)
			if err != nil {
				t.Fatalf("Failed to read meta files: %v", err)
			}
			if err := json.Unmarshal(jsonData, &meta); err != nil {
				t.Fatalf("Failed to unmarshal meta files: %v", err)
			}
			assert.Len(t, meta.Files, 1)
			assert.Equal(t, "artifacts.tar.gz", meta.Archive)
			assert.Equal(t, scraper.DataTypeTarball, meta.DataType)
			assert.Equal(t, "file1", meta.Files[0].Name)
			assert.Equal(t, int64(len(testContent)), meta.Files[0].Size)
			assert.Equal(t, scraper.DataTypeRaw, object.DataType)
		case "artifacts.tar.gz":
			assert.Equal(t, scraper.DataTypeTarball, object.DataType)
		default:
			t.Fatalf("Unexpected object name: %s", object.Name)
		}

		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		assert.Equal(t, "/my/directory/file1", path)
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, []string{".*"}, processFn, notifyFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
	assert.Equal(t, 2, processFnCallCount)
}

func TestRecursiveFilesystemExtractor_ExtractEmpty(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, os.ErrNotExist)
	extractor := scraper.NewRecursiveFilesystemExtractor(fs)

	processFn := func(ctx context.Context, object *scraper.Object) error {
		t.Fatalf("processFn should not be called when no files were scraped")
		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		t.Fatalf("notifyFn should not be called when no files were scraped")
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, []string{".*"}, processFn, notifyFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
}

func TestArchiveFilesystemExtractor_ExtractEmpty(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, os.ErrNotExist)
	extractor := scraper.NewArchiveFilesystemExtractor(fs)

	processFn := func(ctx context.Context, object *scraper.Object) error {
		t.Fatalf("processFn should not be called when no files were scraped")
		return nil
	}

	notifyFn := func(ctx context.Context, path string) error {
		t.Fatalf("notifyFn should not be called when no files were scraped")
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, []string{".*"}, processFn, notifyFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
}
