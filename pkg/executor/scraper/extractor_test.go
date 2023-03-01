package scraper_test

import (
	"bufio"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/common/filesystem"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
)

func TestFilesystemExtractor_Extract(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := filesystem.NewMockFileSystem(ctrl)
	fs.EXPECT().OpenFileBuffered("file1").Return(bufio.NewReader(strings.NewReader("test")), nil)
	extractor := scraper.NewFilesystemExtractor("/my/directory", fs)

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

	// Call the Extract function
	err := extractor.Extract(context.Background(), processFn)
	assert.NoErrorf(t, err, "Extract failed: %v", err)
}
