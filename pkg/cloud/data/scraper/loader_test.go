package scraper_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	cloudscraper "github.com/kubeshop/testkube/pkg/cloud/data/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
)

func TestCloudLoader_Load(t *testing.T) {
	t.Parallel()

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
		meta        map[string]any
		data        io.Reader
		setup       func() *cloudscraper.CloudLoader
		putErr      error
		wantErr     bool
		errContains string
	}{
		{
			name: "valid meta and data",
			meta: map[string]any{
				"executionId":   "my-execution-id",
				"testName":      "my-test",
				"testSuiteName": "my-test-suite",
			},
			data: nil,
			setup: func() *cloudscraper.CloudLoader {
				req := &cloudscraper.PutObjectSignedURLRequest{
					Object:        "my-object",
					ExecutionID:   "my-execution-id",
					TestName:      "my-test",
					TestSuiteName: "my-test-suite",
				}

				mockExecutor.EXPECT().Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req)).Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil).Times(1)
				return cloudscraper.NewCloudLoader(mockExecutor)
			},
			putErr:  nil,
			wantErr: false,
		},
		{
			name: "missing meta",
			meta: map[string]any{
				"object": "my-object",
			},
			data: nil,
			setup: func() *cloudscraper.CloudLoader {
				return cloudscraper.NewCloudLoader(mockExecutor)
			},
			wantErr:     true,
			errContains: "executionId is missing",
		},
		{
			name: "invalid meta",
			meta: map[string]any{
				"object":        "my-object",
				"executionId":   123, // invalid type
				"testName":      "my-test",
				"testSuiteName": "my-test-suite",
			},
			data: nil,
			setup: func() *cloudscraper.CloudLoader {
				return cloudscraper.NewCloudLoader(mockExecutor)
			},
			wantErr:     true,
			errContains: "executionId is not a string",
		},
		{
			name: "executor error",
			meta: map[string]any{
				"executionId":   "my-execution-id",
				"testName":      "my-test",
				"testSuiteName": "my-test-suite",
			},
			data: nil,
			setup: func() *cloudscraper.CloudLoader {
				req := &cloudscraper.PutObjectSignedURLRequest{
					Object:        "my-object",
					ExecutionID:   "my-execution-id",
					TestName:      "my-test",
					TestSuiteName: "my-test-suite",
				}

				mockExecutor.EXPECT().Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req)).Return(nil, errors.New("connection error")).Times(1)
				return cloudscraper.NewCloudLoader(mockExecutor)
			},
			wantErr:     true,
			errContains: "failed to get signed URL for object [my-object]: connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			cloudLoader := tt.setup()
			object := &scraper.Object{
				Name: "my-object",
				Data: tt.data,
			}
			err := cloudLoader.Load(ctx, object, tt.meta)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
