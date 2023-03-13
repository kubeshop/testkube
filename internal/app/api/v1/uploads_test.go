package v1

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/server"
)

func TestTestkubeAPI_UploadCopyFiles(t *testing.T) {
	app := fiber.New()
	storage := MockStorage{}
	storage.GetValidBucketNameFn = func(parentType string, parentName string) string {
		return fmt.Sprintf("%s-%s", parentType, parentName)
	}
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		Storage: &storage,
	}
	route := "/uploads"

	app.Post(route, s.UploadFiles())

	tests := []struct {
		name                string
		parentName          string
		parentType          string
		filePath            string
		fileContent         []byte
		expectedCode        int
		expectedBucketName  string
		expectedFileContent []byte
		expectedObjectSize  int64
	}{
		{
			name:         "no file",
			expectedCode: fiber.StatusBadRequest,
		},
		{
			name:                "file specified on execution",
			parentName:          "1",
			parentType:          "execution",
			filePath:            "/data/file1",
			fileContent:         []byte("first file"),
			expectedCode:        fiber.StatusOK,
			expectedBucketName:  "execution-1",
			expectedFileContent: []byte("first file"),
			expectedObjectSize:  int64(10),
		},
		{
			name:                "file specified on test",
			parentName:          "2",
			parentType:          "test",
			filePath:            "/data/file2",
			fileContent:         []byte("second file"),
			expectedCode:        fiber.StatusOK,
			expectedBucketName:  "test-2",
			expectedFileContent: []byte("second file"),
			expectedObjectSize:  int64(11),
		},
		{
			name:                "file specified on test suite",
			parentName:          "3",
			parentType:          "test-suite",
			filePath:            "/data/file3",
			fileContent:         []byte("third file"),
			expectedCode:        fiber.StatusOK,
			expectedBucketName:  "test-suite-3",
			expectedFileContent: []byte("third file"),
			expectedObjectSize:  int64(10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.UploadFileFn = func(bucket string, filePath string, reader io.Reader, objectSize int64) error {
				assert.Equal(t, tt.expectedBucketName, bucket)
				assert.Equal(t, tt.filePath, filePath)
				assert.Equal(t, tt.expectedObjectSize, objectSize)

				assert.NotEmpty(t, reader)
				file := make([]byte, tt.expectedObjectSize)
				n, err := io.ReadFull(reader, file)
				assert.NoError(t, err)
				assert.Positive(t, n)
				assert.Equal(t, tt.expectedObjectSize, int64(n))
				assert.Equal(t, tt.fileContent, file)

				return nil
			}

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("attachment", filepath.Base(tt.filePath))
			if err != nil {
				t.Error(err)
			}

			if _, err := io.Copy(part, bytes.NewBuffer(tt.fileContent)); err != nil {
				t.Error(err)
			}
			err = writer.WriteField("parentName", tt.parentName)
			if err != nil {
				t.Error(err)
			}
			err = writer.WriteField("parentType", tt.parentType)
			if err != nil {
				t.Error(err)
			}
			err = writer.WriteField("filePath", tt.filePath)
			if err != nil {
				t.Error(err)
			}
			err = writer.Close()
			if err != nil {
				t.Error(err)
			}

			req := httptest.NewRequest(http.MethodPost, route, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode, tt.name)
		})
	}
}

type MockStorage struct {
	UploadFileFn         func(bucket string, filePath string, reader io.Reader, objectSize int64) error
	PlaceFilesFn         func(buckets []string, prefix string) error
	GetValidBucketNameFn func(parentType string, parentName string) string
}

func (m MockStorage) CreateBucket(bucket string) error {
	panic("not implemented")
}

func (m MockStorage) DeleteBucket(bucket string, force bool) error {
	panic("not implemented")
}
func (m MockStorage) ListBuckets() ([]string, error) {
	panic("not implemented")
}
func (m MockStorage) ListFiles(bucketFolder string) ([]testkube.Artifact, error) {
	panic("not implemented")
}
func (m MockStorage) SaveFile(bucketFolder, filePath string) error {
	panic("not implemented")
}
func (m MockStorage) DownloadFile(bucketFolder, file string) (*minio.Object, error) {
	panic("not implemented")
}
func (m MockStorage) UploadFile(bucketFolder string, filePath string, reader io.Reader, objectSize int64) error {
	if m.UploadFileFn == nil {
		panic("not implemented")
	}
	return m.UploadFileFn(bucketFolder, filePath, reader, objectSize)
}

func (m MockStorage) PlaceFiles(buckets []string, prefix string) error {
	if m.PlaceFilesFn == nil {
		panic("not implemented")
	}
	return m.PlaceFilesFn(buckets, prefix)
}

func (m MockStorage) GetValidBucketName(parentType string, parentName string) string {
	if m.GetValidBucketNameFn == nil {
		panic("not implemented")
	}
	return m.GetValidBucketNameFn(parentType, parentName)
}

func (m MockStorage) DeleteFile(bucket, filePath string) error {
	panic("not implemented")
}

func (m MockStorage) ListFilesFromBucket(bucket string) ([]testkube.Artifact, error) {
	panic("not implemented")
}

func (m MockStorage) SaveFileToBucket(bucket, bucketFolder, filePath string) error {
	panic("not implemented")
}

func (m MockStorage) DownloadFileFromBucket(bucket, bucketFolder, file string) (*minio.Object, error) {
	panic("not implemented")
}

func (m MockStorage) UploadFileToBucket(bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	panic("not implemented")
}

func (m MockStorage) DeleteFileFromBucket(bucket, bucketFolder, file string) error {
	panic("not implemented")
}
