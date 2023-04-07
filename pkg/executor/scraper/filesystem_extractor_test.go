package scraper_test

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	filesystem2 "github.com/kubeshop/testkube/pkg/filesystem"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/executor/scraper"
)

func TestRecursiveFilesystemExtractor_Extract(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem2.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, nil)
	fs.EXPECT().OpenFileBuffered("/my/directory/file1").Return(bufio.NewReader(strings.NewReader("test")), nil)
	extractor := scraper.NewRecursiveFilesystemExtractor(fs)

	// Set up the expected calls to the mocked fs object
	fs.EXPECT().Walk("/my/directory", gomock.Any()).Return(nil).DoAndReturn(func(_ string, walkFn filepath.WalkFunc) error {
		fileInfo := filesystem2.MockFileInfo{
			FName:  "file1",
			FIsDir: false,
		}
		return walkFn("/my/directory/file1", &fileInfo, nil)
	})

	processFn := func(ctx context.Context, object *scraper.Object) error {
		assert.Equal(t, "file1", object.Name)
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, processFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
}

//func TestArchiveFilesystemExtractor_Extract(t *testing.T) {
//	t.Parallel()
//
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//
//	fs := filesystem2.NewMockFileSystem(ctrl)
//	fs.EXPECT().Stat("/my/directory").Return(nil, nil)
//	testContent := "test"
//	fs.EXPECT().OpenFileBuffered("/my/directory/file1").Return(bufio.NewReader(strings.NewReader("test")), nil)
//	fs.EXPECT().Stat("/my/directory/file1").Return(nil, nil)
//	extractor := scraper.NewArchiveFilesystemExtractor(fs)
//
//	// Set up the expected calls to the mocked fs object
//	fs.EXPECT().Walk("/my/directory", gomock.Any()).Return(nil).DoAndReturn(func(_ string, walkFn filepath.WalkFunc) error {
//		fileInfo := filesystem2.MockFileInfo{
//			FName:  "file1",
//			FIsDir: false,
//		}
//		return walkFn("/my/directory/file1", &fileInfo, nil)
//	})
//
//	processFn := func(ctx context.Context, object *scraper.Object) error {
//		assert.Equal(t, "archive.tar.gz", object.Name)
//		return nil
//	}
//
//	// Call the Extract function
//	err := extractor.Extract(context.Background(), []string{"/my/directory"}, processFn)
//	assert.NoErrorf(t, err, "Extract failed: %v", err)
//}

func TestRecursiveFilesystemExtractor_ExtractEmpty(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem2.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, os.ErrNotExist)
	extractor := scraper.NewRecursiveFilesystemExtractor(fs)

	processFn := func(ctx context.Context, object *scraper.Object) error {
		t.Fatalf("processFn should not be called when no files were scraped")
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, processFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
}

func TestArchiveFilesystemExtractor_ExtractEmpty(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem2.NewMockFileSystem(ctrl)
	fs.EXPECT().Stat("/my/directory").Return(nil, os.ErrNotExist)
	extractor := scraper.NewArchiveFilesystemExtractor(fs)

	processFn := func(ctx context.Context, object *scraper.Object) error {
		t.Fatalf("processFn should not be called when no files were scraped")
		return nil
	}

	// Call the Extract function
	err := extractor.Extract(context.Background(), []string{"/my/directory"}, processFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
}
