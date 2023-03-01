//go:build integration

package scraper_test

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/kubeshop/testkube/internal/common/filesystem"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	cloudscraper "github.com/kubeshop/testkube/pkg/cloud/data/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCloudScraper(t *testing.T) {
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

	extractor := scraper.NewFilesystemExtractor(tempDir, filesystem.NewOSFileSystem())

	testServerRequests := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/dummy", r.URL.Path)
		testServerRequests++
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	cloudLoader := cloudscraper.NewCloudLoader(mockExecutor)
	req1 := &cloudscraper.PutObjectSignedURLRequest{
		Object:        "file1.txt",
		ExecutionID:   "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	mockExecutor.
		EXPECT().
		Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req1)).
		Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil)

	req2 := &cloudscraper.PutObjectSignedURLRequest{
		Object:        "file2.txt",
		ExecutionID:   "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	mockExecutor.
		EXPECT().
		Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req2)).
		Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil)

	req3 := &cloudscraper.PutObjectSignedURLRequest{
		Object:        "file3.txt",
		ExecutionID:   "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	mockExecutor.
		EXPECT().
		Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req3)).
		Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil)

	meta := map[string]any{
		"executionId":   "my-execution-id",
		"testName":      "my-test",
		"testSuiteName": "my-test-suite",
	}
	s := scraper.NewScraperV2(extractor, cloudLoader)
	err = s.Scrape(context.Background(), meta)
	assert.NoError(t, err)
	assert.Equal(t, 3, testServerRequests)
}
