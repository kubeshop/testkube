package scraper_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/archive"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func TestMinIOUploader_Upload_Tarball_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	// Create a new MinIO uploader with the appropriate configuration
	uploader, err := scraper.NewMinIOUploader(
		"localhost:9000",
		"minio99",
		"minio123",
		"us-east-1",
		"",
		"test-bucket-fsgds",
		false,
		false,
		"",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("failed to create MinIO loader: %v", err)
	}

	files := []*archive.File{
		{Name: "test/testfile.txt", Mode: 0644, Size: 9, ModTime: time.Now(), Data: bytes.NewBufferString("testdata\n")},
	}

	var buf bytes.Buffer
	tarballService := archive.NewTarballService()
	if err = tarballService.Create(&buf, files); err != nil {
		t.Fatalf("error creating tarball: %v", err)
	}

	size := int64(buf.Len())
	// Create a test object to save to MinIO
	testObject := &scraper.Object{
		Name:     "artifacts.tar.gz",
		Data:     &buf,
		Size:     size,
		DataType: scraper.DataTypeTarball,
	}

	execution := testkube.Execution{Id: "test-execution-id"}
	// Call the Upload function to save the object to MinIO
	err = uploader.Upload(context.Background(), testObject, execution)
	if err != nil {
		t.Fatalf("failed to save file to MinIO: %v", err)
	}

	m := minio.NewClient(
		"localhost:9000",
		"minio99",
		"minio123",
		"us-east-1",
		"",
		"test-bucket-fsgds",
	)
	if err := m.Connect(); err != nil {
		t.Fatalf("error conecting to minio: %v", err)
	}
	artifacts, err := m.ListFiles(context.Background(), "test-execution-id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(artifacts))
	assert.Equal(t, "test/testfile.txt", artifacts[0].Name)
	assert.Equal(t, files[0].Size, int64(artifacts[0].Size))
}

func TestMinIOUploader_Upload_Raw_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	// Create a new MinIO loader with the appropriate configuration
	uploader, err := scraper.NewMinIOUploader(
		"localhost:9000",
		"minio99",
		"minio123",
		"us-east-1",
		"",
		"test-bucket-hgfhfg",
		false,
		false,
		"",
		"",
		"",
	)
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
	// Call the Upload function to save the object to MinIO
	err = uploader.Upload(ctx, testObject, execution)
	if err != nil {
		t.Fatalf("failed to save file to MinIO: %v", err)
	}

	m := minio.NewClient(
		"localhost:9000",
		"minio99",
		"minio123",
		"us-east-1",
		"",
		"test-bucket-hgfhfg",
	)
	if err := m.Connect(); err != nil {
		t.Fatalf("error conecting to minio: %v", err)
	}
	artifacts, err := m.ListFiles(context.Background(), "test-execution-id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(artifacts))
	assert.Equal(t, "test-file.txt", artifacts[0].Name)
	assert.Equal(t, size, int64(artifacts[0].Size))
}
