package artifacts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeshop/testkube/pkg/tcl/cloudtcl/data/testworkflow"

	"github.com/golang/mock/gomock"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common/testdata"
	"github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestHandler_CloudUploader(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create a handler that defines how to respond to requests
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	// Create a new HTTP test server
	server := httptest.NewServer(handler)
	defer server.Close() // Close the server when test finishes

	setFilesystemExpectations := func(fs *filesystem.MockFileSystem) {
		fs.
			EXPECT().
			OpenFileRO("test.log").
			Return(filesystem.NewMockFile("test.log", []byte("test")), nil)
		fs.
			EXPECT().
			OpenFileRO("report/junit.xml").
			Return(filesystem.NewMockFile("report/junit.xml", []byte(testdata.BasicJUnit)), nil)

	}
	setDirectPresignedURLExpectations := func(client *executor.MockExecutor) {
		req1 := artifact.PutObjectSignedURLRequest{
			Object:      "test.log",
			ContentType: "application/octet-stream",
		}
		resp1 := artifact.PutObjectSignedURLResponse{
			URL: server.URL,
		}
		resp1Json, _ := json.Marshal(resp1)
		client.EXPECT().Execute(gomock.Any(), artifact.CmdScraperPutObjectSignedURL, gomock.Eq(&req1)).Return(resp1Json, nil)
		req2 := artifact.PutObjectSignedURLRequest{
			Object:      "report/junit.xml",
			ContentType: "application/octet-stream",
		}
		resp2 := artifact.PutObjectSignedURLResponse{
			URL: server.URL,
		}
		resp2Json, _ := json.Marshal(resp2)
		client.EXPECT().Execute(gomock.Any(), artifact.CmdScraperPutObjectSignedURL, gomock.Eq(&req2)).Return(resp2Json, nil)
	}
	setTarPresignedURLExpectations := func(client *executor.MockExecutor) {
		req1 := artifact.PutObjectSignedURLRequest{
			Object:      "artifacts.tar.gz",
			ContentType: "application/octet-stream",
		}
		resp1 := artifact.PutObjectSignedURLResponse{
			URL: server.URL,
		}
		resp1Json, _ := json.Marshal(resp1)
		client.EXPECT().Execute(gomock.Any(), artifact.CmdScraperPutObjectSignedURL, gomock.Eq(&req1)).Return(resp1Json, nil)
	}
	setJUnitPostProcessorExpectations := func(client *executor.MockExecutor) {
		req := testworkflow.ExecutionsAddReportRequest{
			Filepath: "report/junit.xml",
			Report:   []byte(testdata.BasicJUnit),
		}
		client.
			EXPECT().
			Execute(gomock.Any(), testworkflow.CmdTestWorkflowExecutionAddReport, gomock.Eq(&req)).
			Return(nil, nil)
	}

	tests := []struct {
		name                   string
		processor              Processor
		withJUnitPostProcessor bool
		setup                  func(*filesystem.MockFileSystem, *executor.MockExecutor)
	}{
		{
			name:      "direct processor",
			processor: NewDirectProcessor(),
			setup: func(fs *filesystem.MockFileSystem, client *executor.MockExecutor) {
				setDirectPresignedURLExpectations(client)
			},
		},
		{
			name:                   "direct processor with junit post processor",
			processor:              NewDirectProcessor(),
			withJUnitPostProcessor: true,
			setup: func(fs *filesystem.MockFileSystem, client *executor.MockExecutor) {
				setFilesystemExpectations(fs)
				setDirectPresignedURLExpectations(client)
				setJUnitPostProcessorExpectations(client)
			},
		},
		{
			name:      "tar processor",
			processor: NewTarProcessor("artifacts.tar.gz"),
			setup: func(fs *filesystem.MockFileSystem, client *executor.MockExecutor) {
				setTarPresignedURLExpectations(client)
			},
		},
		{
			name:                   "tar processor with junit post processor",
			withJUnitPostProcessor: true,
			processor:              NewTarProcessor("artifacts.tar.gz"),
			setup: func(fs *filesystem.MockFileSystem, client *executor.MockExecutor) {
				setFilesystemExpectations(fs)
				setTarPresignedURLExpectations(client)
				setJUnitPostProcessorExpectations(client)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockFilesystem := filesystem.NewMockFileSystem(mockCtrl)
			mockExecutor := executor.NewMockExecutor(mockCtrl)
			uploader := NewCloudUploader(mockExecutor)
			if tc.setup != nil {
				tc.setup(mockFilesystem, mockExecutor)
			}
			var handlerOpts []HandlerOpts
			if tc.withJUnitPostProcessor {
				pp := NewJUnitPostProcessor(mockFilesystem, mockExecutor)
				handlerOpts = append(handlerOpts, WithPostProcessor(pp))
			}
			handler := NewHandler(uploader, tc.processor, handlerOpts...)
			err := handler.Start()
			if err != nil {
				t.Errorf("error starting handler: %v", err)
			}

			if err := handler.Start(); err != nil {
				t.Fatalf("error starting handler: %v", err)
			}

			testFile1 := filesystem.NewMockFile("test.log", []byte("test"))
			testFile1Stat, err := testFile1.Stat()
			if err != nil {
				t.Fatalf("error getting file1 stat: %v", err)
			}
			testFile2 := filesystem.NewMockFile("report/junit.xml", []byte(testdata.BasicJUnit))
			testFile2Stat, err := testFile2.Stat()
			if err != nil {
				t.Fatalf("error getting file2 stat: %v", err)
			}
			if err := handler.Add("test.log", testFile1, testFile1Stat); err != nil {
				t.Fatalf("error adding file: %v", err)
			}
			if err := handler.Add("report/junit.xml", testFile2, testFile2Stat); err != nil {
				t.Fatalf("error adding file: %v", err)
			}

			if err := handler.End(); err != nil {
				t.Fatalf("error ending handler: %v", err)
			}
		})
	}
}
