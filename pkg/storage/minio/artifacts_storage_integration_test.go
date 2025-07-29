package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
)

var (
	cfg, _ = config.Get()
)

func TestArtifactClient_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	// Create a unique bucket name for this test
	testBucket := fmt.Sprintf("test-bucket-%d", time.Now().UnixNano())

	directMinioClient, err := minio.New(cfg.StorageEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.StorageAccessKeyID, cfg.StorageSecretAccessKey, cfg.StorageToken),
		Secure: cfg.StorageSSL,
	})
	if err != nil {
		t.Fatalf("unable to create direct minio client: %v", err)
	}

	// Create the test bucket
	err = directMinioClient.MakeBucket(ctx, testBucket, minio.MakeBucketOptions{})
	if err != nil {
		t.Fatalf("unable to create test bucket: %v", err)
	}

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		// Remove bucket and all its contents
		err := directMinioClient.RemoveBucketWithOptions(ctx, testBucket, minio.RemoveBucketOptions{
			ForceDelete: true,
		})
		if err != nil {
			t.Logf("error removing test bucket: %v", err)
		}
	})

	// Prepare MinIO client with test bucket
	minioClient := NewClient(cfg.StorageEndpoint, cfg.StorageAccessKeyID, cfg.StorageSecretAccessKey, cfg.StorageRegion, cfg.StorageToken, testBucket)
	if err := minioClient.Connect(); err != nil {
		t.Fatalf("unable to connect to minio: %v", err)
	}

	// Create the ArtifactClient
	artifactClient := NewMinIOArtifactClient(minioClient)

	// Test ListFiles
	t.Run("ListFiles", func(t *testing.T) {
		t.Parallel()
		// Upload a test file
		_, err = directMinioClient.PutObject(ctx, testBucket, "test-execution-id-1/test-file", bytes.NewReader([]byte("test-content")), 12, minio.PutObjectOptions{})
		if err != nil {
			t.Fatalf("unable to upload file: %v", err)
		}
		// Call ListFiles
		files, err := artifactClient.ListFiles(ctx, "test-execution-id-1", "", "", "")
		assert.NoError(t, err)

		assert.Lenf(t, files, 1, "expected 1 file to be returned")
		assert.Equal(t, "test-file", files[0].Name, "expected file name to be test-file")
		assert.Equal(t, int32(12), files[0].Size, "expected file size to be 11")
	})

	// Test DownloadFile
	t.Run("DownloadFile", func(t *testing.T) {
		t.Parallel()
		// Upload a test file
		_, err = directMinioClient.PutObject(ctx, testBucket, "test-execution-id-2/test-file", bytes.NewReader([]byte("test-content")), 12, minio.PutObjectOptions{})
		if err != nil {
			t.Fatalf("unable to upload file: %v", err)
		}

		reader, err := artifactClient.DownloadFile(ctx, "test-file", "test-execution-id-2", "", "", "")
		if err != nil {
			t.Fatalf("unable to download file: %v", err)
		}

		content, err := io.ReadAll(reader)
		assert.NoError(t, err)

		assert.Equalf(t, "test-content", string(content), "downloaded file content does not match expected content")
	})

	t.Run("UploadFile", func(t *testing.T) {
		t.Parallel()

		err = artifactClient.UploadFile(ctx, "test-execution-id-3", "test-file", bytes.NewReader([]byte("test-content")), 12)
		if err != nil {
			t.Fatalf("error uploading artifact file: %v", err)
		}

		// Check if the file is uploaded
		obj, err := directMinioClient.GetObject(ctx, testBucket, "test-execution-id-3/test-file", minio.GetObjectOptions{})
		if err != nil {
			t.Fatalf("unable to get object from minio: %v", err)
		}
		stat, err := obj.Stat()
		if err != nil {
			t.Fatalf("unable to get object stat from minio: %v", err)
		}
		assert.Equal(t, int64(12), stat.Size, "expected file size to be 12")
	})

	t.Run("PlaceFiles", func(t *testing.T) {
		t.Parallel()
		// Create a temporary directory to place files
		tempDir, err := os.MkdirTemp("", "test-artifactclient")
		if err != nil {
			t.Fatalf("unable to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)
		// Upload test files
		_, err = directMinioClient.PutObject(ctx, testBucket, "test-execution-id-4/test-file1", bytes.NewReader([]byte("test-content")), 12, minio.PutObjectOptions{})
		if err != nil {
			t.Fatalf("unable to upload file: %v", err)
		}
		_, err = directMinioClient.PutObject(ctx, testBucket, "test-execution-id-4/test-file2", bytes.NewReader([]byte("test-content")), 12, minio.PutObjectOptions{})
		if err != nil {
			t.Fatalf("unable to upload file: %v", err)
		}

		err = artifactClient.PlaceFiles(ctx, []string{"test-execution-id-4"}, tempDir)
		if err != nil {
			t.Fatalf("error placing files: %v", err)
		}

		// Check if the files are placed in the temporary directory
		content, err := os.ReadFile(tempDir + "/" + "test-file1")
		if err != nil {
			t.Fatalf("unable to read file: %v", err)
		}
		assert.Equal(t, "test-content", string(content), "placed file content does not match expected content")
		content, err = os.ReadFile(tempDir + "/" + "test-file2")
		if err != nil {
			t.Fatalf("unable to read file: %v", err)
		}
		assert.Equal(t, "test-content", string(content), "placed file content does not match expected content")
	})
}
