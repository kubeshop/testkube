package v1

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

func TestTestkubeAPI_UploadCopyFiles(t *testing.T) {
	app := fiber.New()
	storage := MockStorage{}
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		Storage: &storage,
	}

	app.Post("/executions/:id/copyFiles/:filename", s.UploadCopyFiles())
	app.Post("/tests/:id/copyFiles/:filename", s.UploadCopyFiles())
	app.Post("/test-suites/:id/copyFiles/:filename", s.UploadCopyFiles())

	tests := []struct {
		name               string
		route              string
		expectedCode       int
		expectedBucketName string
		expectedObjectSize int64
		ownerID            string // ID of the execution / test / test suite the file belongs to
		filePath           string
		fileContent        []byte
	}{
		{
			name:         "no file",
			route:        "/executions/1/copyFiles/noFile",
			expectedCode: fiber.StatusBadRequest,
		},
		{
			name:               "file specified on execution",
			route:              "/executions/1/copyFiles/file1",
			expectedCode:       fiber.StatusOK,
			expectedBucketName: "execution-1",
			expectedObjectSize: 10,
			ownerID:            "1",
			fileContent:        []byte("first file"),
			filePath:           "file1",
		},
		{
			name:               "file specified on test",
			route:              "/tests/2/copyFiles/file2",
			expectedCode:       fiber.StatusOK,
			expectedBucketName: "test-2",
			expectedObjectSize: 11,
			ownerID:            "2",
			fileContent:        []byte("second file"),
			filePath:           "file2",
		},
		{
			name:               "file specified on test suite",
			route:              "/test-suites/3/copyFiles/file3",
			expectedCode:       fiber.StatusOK,
			expectedBucketName: "test-suite-3",
			expectedObjectSize: 10,
			ownerID:            "3",
			fileContent:        []byte("third file"),
			filePath:           "file3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.SaveCopyFileFn = func(bucket string, filePath string, reader io.Reader, objectSize int64) error {
				assert.Equal(t, tt.expectedBucketName, bucket)
				assert.Equal(t, tt.filePath, filePath)
				assert.Equal(t, tt.expectedObjectSize, objectSize)
				// TODO fix
				// assert.NotEmpty(t, reader)
				// file := make([]byte, tt.expectedObjectSize)
				// n, err := io.ReadFull(reader, file)
				// assert.NoError(t, err)
				// assert.Equal(t, tt.expectedObjectSize, n)
				// assert.Equal(t, tt.fileContent, file)

				return nil
			}

			req := httptest.NewRequest("POST", tt.route, bytes.NewReader(tt.fileContent))
			req.Header.Set("Content-Type", "application/octet-stream")

			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode, tt.name)
		})
	}
}

type MockStorage struct {
	SaveCopyFileFn func(bucket string, filePath string, reader io.Reader, objectSize int64) error
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
func (m MockStorage) ListFiles(bucket string) ([]testkube.Artifact, error) {
	panic("not implemented")
}
func (m MockStorage) SaveFile(bucket, filePath string) error {
	panic("not implemented")
}
func (m MockStorage) DownloadFile(bucket, file string) (*minio.Object, error) {
	panic("not implemented")
}
func (m MockStorage) SaveCopyFile(bucket string, filePath string, reader io.Reader, objectSize int64) error {
	if m.SaveCopyFileFn == nil {
		panic("not implemented")
	}
	return m.SaveCopyFileFn(bucket, filePath, reader, objectSize)
}
