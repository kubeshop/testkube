package artifacts

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common/testdata"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestHandler_CloudUploader(t *testing.T) {
	// Populate empty internal configuration, as it is required for the Toolkit
	_ = os.Setenv("TK_CFG", "{}")

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
			OpenFileRO("/test.log").
			Return(filesystem.NewMockFile("test.log", []byte("test")), nil)
		fs.
			EXPECT().
			OpenFileRO("/report/junit.xml").
			Return(filesystem.NewMockFile("report/junit.xml", []byte(testdata.BasicJUnit)), nil)

	}
	setDirectPresignedURLExpectations := func(client *controlplaneclient.MockClient) {
		client.EXPECT().
			SaveExecutionArtifactGetPresignedURL(gomock.Any(), "env123", "exec123", "workflow123", "step123", "test.log", "application/octet-stream").
			Return(server.URL, nil)
		client.EXPECT().
			SaveExecutionArtifactGetPresignedURL(gomock.Any(), "env123", "exec123", "workflow123", "step123", "report/junit.xml", "application/octet-stream").
			Return(server.URL, nil)
	}
	setTarPresignedURLExpectations := func(client *controlplaneclient.MockClient) {
		client.EXPECT().
			SaveExecutionArtifactGetPresignedURL(gomock.Any(), "env123", "exec123", "workflow123", "step123", "artifacts.tar.gz", "application/octet-stream").
			Return(server.URL, nil)
	}
	setJUnitPostProcessorExpectations := func(client *controlplaneclient.MockClient) {
		client.EXPECT().
			AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "report/junit.xml", []byte(testdata.BasicJUnit)).
			Return(nil)
	}

	tests := []struct {
		name                   string
		processor              Processor
		withJUnitPostProcessor bool
		setup                  func(*filesystem.MockFileSystem, *controlplaneclient.MockClient)
	}{
		{
			name:      "direct processor",
			processor: NewDirectProcessor(),
			setup: func(fs *filesystem.MockFileSystem, client *controlplaneclient.MockClient) {
				setDirectPresignedURLExpectations(client)
			},
		},
		{
			name:                   "direct processor with junit post processor",
			processor:              NewDirectProcessor(),
			withJUnitPostProcessor: true,
			setup: func(fs *filesystem.MockFileSystem, client *controlplaneclient.MockClient) {
				setFilesystemExpectations(fs)
				setDirectPresignedURLExpectations(client)
				setJUnitPostProcessorExpectations(client)
			},
		},
		{
			name:      "tar processor",
			processor: NewTarProcessor("artifacts.tar.gz"),
			setup: func(fs *filesystem.MockFileSystem, client *controlplaneclient.MockClient) {
				setTarPresignedURLExpectations(client)
			},
		},
		{
			name:                   "tar processor with junit post processor",
			withJUnitPostProcessor: true,
			processor:              NewTarProcessor("artifacts.tar.gz"),
			setup: func(fs *filesystem.MockFileSystem, client *controlplaneclient.MockClient) {
				setFilesystemExpectations(fs)
				setTarPresignedURLExpectations(client)
				setJUnitPostProcessorExpectations(client)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockFilesystem := filesystem.NewMockFileSystem(mockCtrl)
			mockClient := controlplaneclient.NewMockClient(mockCtrl)
			uploader := NewCloudUploader(mockClient, "env123", "exec123", "workflow123", "step123")
			if tc.setup != nil {
				tc.setup(mockFilesystem, mockClient)
			}
			var handlerOpts []HandlerOpts
			if tc.withJUnitPostProcessor {
				pp := NewJUnitPostProcessor(mockFilesystem, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
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
