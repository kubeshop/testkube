package commands

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/kubeshop/testkube/pkg/tcl/cloudtcl/data/testworkflow"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

//go:embed testdata/*
var testDataFixtures embed.FS

func TestRun_Integration(t *testing.T) {
	test.IntegrationTest(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create a handler that defines how to respond to requests
	httpRequestCount := 0
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpRequestCount++
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	// Create a new HTTP test server
	server := httptest.NewServer(h)
	defer server.Close()

	walker, err := artifacts.CreateWalker([]string{"./testdata/*"}, []string{"/"}, "/")
	if err != nil {
		t.Fatalf("failed to create walker: %v", err)
	}
	processor := artifacts.NewDirectProcessor()
	mockClient := executor.NewMockExecutor(mockCtrl)
	mockResponse := artifact.PutObjectSignedURLResponse{
		URL: server.URL,
	}
	mockResponseJson, _ := json.Marshal(mockResponse)
	mockClient.EXPECT().Execute(gomock.Any(), artifact.CmdScraperPutObjectSignedURL, gomock.Any()).Return(mockResponseJson, nil).Times(2)
	mockClient.EXPECT().Execute(gomock.Any(), testworkflow.CmdTestWorkflowExecutionAddReport, gomock.Any()).Return(nil, nil)
	uploader := artifacts.NewCloudUploader(mockClient)
	postProcessor := artifacts.NewJUnitPostProcessor(filesystem.NewOSFileSystem(), mockClient)
	handler := artifacts.NewHandler(uploader, processor, artifacts.WithPostProcessor(postProcessor))

	run(handler, walker, testDataFixtures)

	assert.Equal(t, 2, httpRequestCount)
}
