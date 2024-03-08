package v1

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/kubeshop/testkube/pkg/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/server"
)

func TestTestkubeAPI_UploadCopyFiles(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockArtifactsStorage := storage.NewMockArtifactsStorage(mockCtrl)

	app := fiber.New()
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		ArtifactsStorage: mockArtifactsStorage,
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
		setupMocks          func()
	}{
		{
			name:         "no file",
			expectedCode: fiber.StatusBadRequest,
			setupMocks:   func() {},
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
			setupMocks: func() {
				mockArtifactsStorage.EXPECT().GetValidBucketName("execution", "1").Return("execution-1")
				mockArtifactsStorage.EXPECT().UploadFile(gomock.Any(), "execution-1", "/data/file1", gomock.Any(), int64(10)).Return(nil)
			},
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
			setupMocks: func() {
				mockArtifactsStorage.EXPECT().GetValidBucketName("test", "2").Return("test-2")
				mockArtifactsStorage.EXPECT().UploadFile(gomock.Any(), "test-2", "/data/file2", gomock.Any(), int64(11)).Return(nil)
			},
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
			setupMocks: func() {
				mockArtifactsStorage.EXPECT().GetValidBucketName("test-suite", "3").Return("test-suite-3")
				mockArtifactsStorage.EXPECT().UploadFile(gomock.Any(), "test-suite-3", "/data/file3", gomock.Any(), int64(10)).Return(nil)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.setupMocks()
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("attachment", filepath.Base(tt.filePath))
			if err != nil {
				t.Error(err)
			}

			if _, err = io.Copy(part, bytes.NewBuffer(tt.fileContent)); err != nil {
				t.Error(err)
			}

			if err = writer.WriteField("parentName", tt.parentName); err != nil {
				t.Error(err)
			}

			if err = writer.WriteField("parentType", tt.parentType); err != nil {
				t.Error(err)
			}

			if err = writer.WriteField("filePath", tt.filePath); err != nil {
				t.Error(err)
			}

			if err = writer.Close(); err != nil {
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
