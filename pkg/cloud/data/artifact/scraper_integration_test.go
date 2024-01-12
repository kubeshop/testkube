package artifact_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	cloudscraper "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestCloudScraper_ArchiveFilesystemExtractor_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(tempDir)

	err = os.Mkdir(filepath.Join(tempDir, "subdir"), os.ModePerm)
	assert.NoError(t, err)

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	file3 := filepath.Join(tempDir, "subdir", "file3.txt")

	err = os.WriteFile(file1, []byte("test1"), os.ModePerm)
	assert.NoError(t, err)

	err = os.WriteFile(file2, []byte("test2"), os.ModePerm)
	assert.NoError(t, err)

	err = os.WriteFile(file3, []byte("test3"), os.ModePerm)
	assert.NoError(t, err)

	extractor := scraper.NewArchiveFilesystemExtractor(filesystem.NewOSFileSystem(), scraper.GenerateTarballMetaFile())

	testServerRequests := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/dummy", r.URL.Path)
		testServerRequests++
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	cloudLoader := cloudscraper.NewCloudUploader(mockExecutor, false)
	req := &cloudscraper.PutObjectSignedURLRequest{
		Object:        "artifacts.tar.gz",
		ExecutionID:   "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	mockExecutor.
		EXPECT().
		Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req)).
		Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil)
	req2 := &cloudscraper.PutObjectSignedURLRequest{
		Object:        ".testkube-meta-files.json",
		ExecutionID:   "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	mockExecutor.
		EXPECT().
		Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req2)).
		Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil)

	// given
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := cloudevents.NewEventFromHTTPRequest(r)
		// then
		assert.NoError(t, err)
	})

	svr := httptest.NewServer(testHandler)
	defer svr.Close()

	client, err := cloudevents.NewClientHTTP(cloudevents.WithTarget(svr.URL))
	assert.NoError(t, err)

	s := scraper.NewExtractLoadScraper(extractor, cloudLoader, client, "", "")
	execution := testkube.Execution{
		Id:            "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	err = s.Scrape(context.Background(), []string{tempDir}, []string{".*"}, execution)

	assert.NoError(t, err)
	assert.Equal(t, 2, testServerRequests)
}

func TestCloudScraper_RecursiveFilesystemExtractor_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(tempDir)

	err = os.Mkdir(filepath.Join(tempDir, "subdir"), os.ModePerm)
	assert.NoError(t, err)

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	file3 := filepath.Join(tempDir, "subdir", "file3.txt")

	err = os.WriteFile(file1, []byte("test1"), os.ModePerm)
	assert.NoError(t, err)

	err = os.WriteFile(file2, []byte("test2"), os.ModePerm)
	assert.NoError(t, err)

	err = os.WriteFile(file3, []byte("test3"), os.ModePerm)
	assert.NoError(t, err)

	extractor := scraper.NewRecursiveFilesystemExtractor(filesystem.NewOSFileSystem())

	testServerRequests := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/dummy", r.URL.Path)
		testServerRequests++
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	cloudLoader := cloudscraper.NewCloudUploader(mockExecutor, false)
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
		Object:        "subdir/file3.txt",
		ExecutionID:   "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	mockExecutor.
		EXPECT().
		Execute(gomock.Any(), cloudscraper.CmdScraperPutObjectSignedURL, gomock.Eq(req3)).
		Return([]byte(`{"URL":"`+testServer.URL+`/dummy"}`), nil)

	// given
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := cloudevents.NewEventFromHTTPRequest(r)
		// then
		assert.NoError(t, err)
	})

	svr := httptest.NewServer(testHandler)
	defer svr.Close()

	client, err := cloudevents.NewClientHTTP(cloudevents.WithTarget(svr.URL))
	assert.NoError(t, err)

	s := scraper.NewExtractLoadScraper(extractor, cloudLoader, client, "", "")
	execution := testkube.Execution{
		Id:            "my-execution-id",
		TestName:      "my-test",
		TestSuiteName: "my-test-suite",
	}
	err = s.Scrape(context.Background(), []string{tempDir}, []string{".*"}, execution)
	assert.NoError(t, err)
	assert.Equal(t, 3, testServerRequests)
}
