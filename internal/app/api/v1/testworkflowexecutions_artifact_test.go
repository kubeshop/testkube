package v1

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/storage"
)

// TestGetArtifactWithSlashedFilename verifies that fetching an artifact whose
// name contains forward slashes (e.g. "results/junit.xml") works end-to-end
// through the real Fiber route → handler → storage chain.
//
// Before the fix, the route used /:filename (single-segment), so Fiber couldn't
// match the decoded path and the agent returned no response (causing a 408 timeout
// at the cloud API level).
func TestGetArtifactWithSlashedFilename(t *testing.T) {
	t.Parallel()

	const (
		execID  = "exec-1"
		wfName  = "my-workflow"
		content = "<testsuites/>"
	)

	tests := []struct {
		name        string
		filename    string // decoded filename as stored in artifact storage
		urlFilename string // filename as it appears in the URL (%2F for /)
	}{
		{
			name:        "simple filename",
			filename:    "report.xml",
			urlFilename: "report.xml",
		},
		{
			name:        "filename with one subdirectory",
			filename:    "results/junit.xml",
			urlFilename: "results%2Fjunit.xml",
		},
		{
			name:        "filename with nested subdirectories",
			filename:    "a/b/c/report.xml",
			urlFilename: "a%2Fb%2Fc%2Freport.xml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockResults := testworkflow.NewMockRepository(ctrl)
			mockStorage := storage.NewMockArtifactsStorage(ctrl)

			mockResults.EXPECT().Get(gomock.Any(), execID).Return(testkube.TestWorkflowExecution{
				Id:       execID,
				Workflow: &testkube.TestWorkflow{Name: wfName},
			}, nil)

			mockStorage.EXPECT().
				DownloadFile(gomock.Any(), tc.filename, execID, "", "", wfName).
				Return(io.NopCloser(strings.NewReader(content)), nil)

			testAPI := &TestkubeAPI{
				TestWorkflowResults: mockResults,
				ArtifactsStorage:    mockStorage,
				Log:                 log.DefaultLogger,
			}

			app := fiber.New()
			execGroup := app.Group("/v1/test-workflow-executions")
			execGroup.Get("/:executionID/artifacts/:filename+", testAPI.GetTestWorkflowArtifactHandler())

			url := "/v1/test-workflow-executions/" + execID + "/artifacts/" + tc.urlFilename
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode,
				"expected 200 but got %d — if no response, the route didn't match the slashed filename", resp.StatusCode)
			assert.Equal(t, content, string(body))
		})
	}
}
