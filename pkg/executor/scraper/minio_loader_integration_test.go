//go:build integration

package scraper_test

import (
	"context"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestMinIOLoader_Load(t *testing.T) {
	t.Parallel()

	m := minio.NewClient("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket", false)
	if err := m.Connect(); err != nil {
		t.Fatalf("error conecting to minio: %v", err)
	}

	// Create a new MinIO loader with the appropriate configuration
	loader, err := scraper.NewMinIOLoader("localhost:9000", "minio99", "minio123", "us-east-1", "", "test-bucket", false)
	if err != nil {
		t.Fatalf("failed to create MinIO loader: %v", err)
	}

	// Create a test context
	ctx := context.Background()

	// Create a test object to save to MinIO
	size := int64(len("test data"))
	testObject := &scraper.Object{
		Name: "test-file.txt",
		Data: strings.NewReader("test data"),
		Size: size,
	}

	// Create a test metadata map with an execution ID
	testMeta := map[string]interface{}{
		"executionId": "test-execution-id",
	}

	// Call the Load function to save the object to MinIO
	err = loader.Load(ctx, testObject, testMeta)
	if err != nil {
		t.Fatalf("failed to save file to MinIO: %v", err)
	}

	buckets, err := m.ListBuckets()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(buckets))
	assert.Equal(t, "test-bucket", buckets[0])

	artifacts, err := m.ListFiles("test-execution-id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(artifacts))
	assert.Equal(t, "test-file.txt", artifacts[0].Name)
	assert.Equal(t, int32(size), artifacts[0].Size)
}
