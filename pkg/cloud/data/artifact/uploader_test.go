package artifact_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/scraper/scrapertypes"

	"github.com/stretchr/testify/assert"

	"github.com/pkg/errors"
	gomock "go.uber.org/mock/gomock"

	cloudscraper "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

func TestCloudLoader_Load(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/dummy", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	tests := []struct {
		name        string
		execution   testkube.Execution
		data        io.Reader
		setup       func() *cloudscraper.CloudUploader
		putErr      error
		wantErr     bool
		errContains string
	}{
		{
			name: "valid data",
			execution: testkube.Execution{
				Id:            "my-execution-id",
				TestName:      "my-test",
				TestSuiteName: "my-test-suite",
			},
			data: nil,
			setup: func() *cloudscraper.CloudUploader {
				req := &cloudscraper.PutObjectSignedURLRequest{
					Object:        "my-object",
					ExecutionID:   "my-execution-id",
					TestName:      "my-test",
					TestSuiteName: "my-test-suite",
					ContentType:   "text/plain",
				}

				mockExecutor.EXPECT().Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req)).Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil).Times(1)
				return cloudscraper.NewCloudUploader(mockExecutor, false)
			},
			putErr:  nil,
			wantErr: false,
		},
		{
			name: "executor error",
			execution: testkube.Execution{
				Id:            "my-execution-id",
				TestName:      "my-test",
				TestSuiteName: "my-test-suite",
			},
			data: nil,
			setup: func() *cloudscraper.CloudUploader {
				req := &cloudscraper.PutObjectSignedURLRequest{
					Object:        "my-object",
					ExecutionID:   "my-execution-id",
					TestName:      "my-test",
					TestSuiteName: "my-test-suite",
					ContentType:   "text/plain",
				}

				mockExecutor.EXPECT().Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req)).Return(nil, errors.New("connection error")).Times(1)
				return cloudscraper.NewCloudUploader(mockExecutor, false)
			},
			wantErr:     true,
			errContains: "failed to get signed URL for object [my-object]: connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			cloudLoader := tt.setup()
			object := &scrapertypes.Object{
				Name: "my-object",
				Data: tt.data,
			}
			err := cloudLoader.Upload(ctx, object, tt.execution)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
