package commands

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/mapper/cdevents"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

//go:embed testdata/*
var testDataFixtures embed.FS

func TestArtifactsHandlerRun(t *testing.T) {
	// Populate empty internal configuration, as it is required for the Toolkit
	_ = os.Setenv("TK_CFG", "{}")

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
		if r.Method == http.MethodPost {
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
	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	mockClient.EXPECT().
		SaveExecutionArtifactGetPresignedURL(gomock.Any(), "env123", "exec123", "workflow123", "step123", gomock.Any(), gomock.Any()).
		Return(server.URL, nil).
		Times(2)
	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", gomock.Any(), gomock.Any()).
		Return(nil)
	uploader := artifacts.NewCloudUploader(mockClient, "env123", "exec123", "workflow123", "step123")
	mockFs := filesystem.NewMockFileSystem(mockCtrl)
	mockFs.
		EXPECT().
		OpenFileRO(gomock.Any()).
		AnyTimes().
		DoAndReturn(func(path string) (fs.File, error) {
			b, err := testDataFixtures.ReadFile(path[1:])
			if err != nil {
				return nil, err
			}
			return filesystem.NewMockFile(path[1:], b), nil
		})
	postProcessor := artifacts.NewJUnitPostProcessor(mockFs, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	handler := artifacts.NewHandler(
		uploader,
		processor,
		artifacts.WithPostProcessor(postProcessor),
		artifacts.WithCDEventsTarget(server.URL),
		artifacts.WithCDEventsArtifactParameters(cdevents.CDEventsArtifactParameters{
			Id:           "1",
			Name:         "test-1",
			WorkflowName: "test",
			ClusterID:    "12345",
		}),
	)

	run(handler, walker, testDataFixtures)

	assert.Equal(t, 4, httpRequestCount)
}
