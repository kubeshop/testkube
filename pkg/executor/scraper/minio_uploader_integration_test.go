//go:build integration

package scraper_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/archive"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func TestMinIOLoader_Upload_Tarball(t *testing.T) {
	t.Parallel()

	m := minio.NewClient("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket-1", false)
	if err := m.Connect(); err != nil {
		t.Fatalf("error conecting to minio: %v", err)
	}

	// Create a new MinIO loader with the appropriate configuration
	loader, err := scraper.NewMinIOUploader("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket-1", false)
	if err != nil {
		t.Fatalf("failed to create MinIO loader: %v", err)
	}

	files := []*archive.File{
		{Name: "testfile.txt", Mode: 0644, Size: 9, ModTime: time.Now(), Data: bytes.NewBufferString("testdata\n")},
	}

	var buf bytes.Buffer
	tarballService := archive.NewTarballService()
	_, err = tarballService.Create(&buf, files)
	if err != nil {
		t.Fatalf("error creating tarball: %v", err)
	}

	size := int64(buf.Len())
	// Create a test object to save to MinIO
	testObject := &scraper.Object{
		Name:     "test-file.txt",
		Data:     &buf,
		Size:     size,
		DataType: scraper.DataTypeTarball,
	}

	execution := testkube.Execution{Id: "test-execution-id"}
	// Call the Upload function to save the object to MinIO
	err = loader.Upload(context.Background(), testObject, execution)
	if err != nil {
		t.Fatalf("failed to save file to MinIO: %v", err)
	}

	artifacts, err := m.ListFiles(context.Background(), "test-execution-id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(artifacts))
	assert.Equal(t, "test-file.txt", artifacts[0].Name)
	assert.Equal(t, size, artifacts[0].Size)
}

func TestMinIOLoader_Upload_Raw(t *testing.T) {
	t.Parallel()

	m := minio.NewClient("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket-1", false)
	if err := m.Connect(); err != nil {
		t.Fatalf("error conecting to minio: %v", err)
	}

	// Create a new MinIO loader with the appropriate configuration
	loader, err := scraper.NewMinIOUploader("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket-1", false)
	if err != nil {
		t.Fatalf("failed to create MinIO loader: %v", err)
	}

	// Create a test context
	ctx := context.Background()

	// Create a test object to save to MinIO
	size := int64(len("test data"))
	testObject := &scraper.Object{
		Name:     "test-file.txt",
		Data:     strings.NewReader("test data"),
		Size:     size,
		DataType: scraper.DataTypeRaw,
	}

	execution := testkube.Execution{Id: "test-execution-id"}
	// Call the Load function to save the object to MinIO
	err = loader.Upload(ctx, testObject, execution)
	if err != nil {
		t.Fatalf("failed to save file to MinIO: %v", err)
	}

	artifacts, err := m.ListFiles(context.Background(), "test-execution-id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(artifacts))
	assert.Equal(t, "test-file.txt", artifacts[0].Name)
	assert.Equal(t, int32(size), artifacts[0].Size)
}
