//go:build integration

package scraper_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func TestMinIOScraper(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

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

	extractor := scraper.NewRecursiveFilesystemExtractor(filesystem.NewOSFileSystem())

	loader, err := scraper.NewMinIOUploader("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket-2", false)
	if err != nil {
		t.Fatalf("error creating minio loader: %v", err)
	}

	execution := testkube.Execution{Id: "minio-test"}
	s := scraper.NewExtractLoadScraper(extractor, loader)
	err = s.Scrape(context.Background(), []string{tempDir}, execution)
	if err != nil {
		t.Fatalf("error scraping: %v", err)
	}

	c := minio.NewClient("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket-2", false)
	assert.NoError(t, c.Connect())
	artifacts, err := c.ListFiles(context.Background(), "test-bucket-2")
	if err != nil {
		t.Fatalf("error listing files from bucket: %v", err)
	}
	assert.True(t, containsArtifact(t, artifacts, "minio-test/file1.txt"))
	assert.True(t, containsArtifact(t, artifacts, "minio-test/file2.txt"))
	assert.True(t, containsArtifact(t, artifacts, "minio-test/subdir/file3.txt"))
}

func containsArtifact(t *testing.T, artifacts []testkube.Artifact, name string) bool {
	t.Helper()

	for _, a := range artifacts {
		if a.Name == name {
			return true
		}
	}
	return false
}
