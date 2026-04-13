package v1

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func TestWriteTarEntry(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	data := []byte(`{"id":"exec-1"}`)
	err := writeTarEntry(tw, "executions/exec-1.json", data)
	require.NoError(t, err)

	data2 := []byte("log line 1\nlog line 2\n")
	err = writeTarEntry(tw, "logs/exec-1.log", data2)
	require.NoError(t, err)

	require.NoError(t, tw.Close())

	// Read the tar archive and verify entries
	tr := tar.NewReader(&buf)

	files := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		content, err := io.ReadAll(tr)
		require.NoError(t, err)
		files[header.Name] = string(content)

		assert.Equal(t, int64(0o644), header.Mode)
	}

	assert.Contains(t, files, "executions/exec-1.json")
	assert.Contains(t, files, "logs/exec-1.log")
	assert.Equal(t, `{"id":"exec-1"}`, files["executions/exec-1.json"])
	assert.Equal(t, "log line 1\nlog line 2\n", files["logs/exec-1.log"])
}

func TestWriteTarEntry_EmptyData(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := writeTarEntry(tw, "empty.txt", []byte{})
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	tr := tar.NewReader(&buf)
	header, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, "empty.txt", header.Name)
	assert.Equal(t, int64(0), header.Size)
}

func TestSequenceEntryMarshal(t *testing.T) {
	entries := []sequenceEntry{
		{WorkflowName: "workflow-a", Number: 10},
		{WorkflowName: "workflow-b", Number: 25},
	}

	data, err := json.Marshal(entries)
	require.NoError(t, err)

	var loaded []sequenceEntry
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Len(t, loaded, 2)

	seqMap := map[string]int32{}
	for _, e := range loaded {
		seqMap[e.WorkflowName] = e.Number
	}
	assert.Equal(t, int32(10), seqMap["workflow-a"])
	assert.Equal(t, int32(25), seqMap["workflow-b"])
}

func TestWriteTarEntry_GzipCompression(t *testing.T) {
	// Verify tar entries work inside gzip stream
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := writeTarEntry(tw, "test.json", []byte(`{"key":"value"}`))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	// Decompress and verify
	gr, err := gzip.NewReader(&buf)
	require.NoError(t, err)
	defer func() { require.NoError(t, gr.Close()) }()

	tr := tar.NewReader(gr)
	header, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, "test.json", header.Name)

	content, err := io.ReadAll(tr)
	require.NoError(t, err)
	assert.Equal(t, `{"key":"value"}`, string(content))
}

// --- Handler integration tests ---

func setupExportTestServer(t *testing.T, maxSize int) (*fiber.App, *gomock.Controller, *testworkflow2.MockRepository, *testworkflow2.MockOutputRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockRepo := testworkflow2.NewMockRepository(ctrl)
	mockOutput := testworkflow2.NewMockOutputRepository(ctrl)

	s := &TestkubeAPI{
		Log:                  log.DefaultLogger,
		TestWorkflowResults:  mockRepo,
		TestWorkflowOutput:   mockOutput,
		exportArchiveMaxSize: maxSize,
	}

	app := fiber.New()
	app.Get("/export", s.ExportExecutionsHandler())

	return app, ctrl, mockRepo, mockOutput
}

func TestExportExecutionsHandler_InvalidSinceDate(t *testing.T) {
	app, ctrl, _, _ := setupExportTestServer(t, 100*1024*1024)
	defer ctrl.Finish()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export?since=not-a-date", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "invalid 'since' date")
}

func TestExportExecutionsHandler_EmptyExport(t *testing.T) {
	app, ctrl, mockRepo, _ := setupExportTestServer(t, 100*1024*1024)
	defer ctrl.Finish()

	mockRepo.EXPECT().GetExecutions(gomock.Any(), gomock.Any()).Return(nil, nil)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), apiclient.ExportArchiveFileName)

	// Verify it's a valid gzip archive with a sequences.json entry
	body, _ := io.ReadAll(resp.Body)
	gr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	var entries []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		entries = append(entries, header.Name)
	}
	assert.Contains(t, entries, "sequences.json")
}

func TestExportExecutionsHandler_WithExecutions(t *testing.T) {
	app, ctrl, mockRepo, mockOutput := setupExportTestServer(t, 100*1024*1024)
	defer ctrl.Finish()

	executions := []testkube.TestWorkflowExecution{
		{
			Id:     "exec-1",
			Number: 5,
			Workflow: &testkube.TestWorkflow{
				Name: "workflow-a",
			},
		},
		{
			Id:     "exec-2",
			Number: 3,
			Workflow: &testkube.TestWorkflow{
				Name: "workflow-a",
			},
		},
	}

	mockRepo.EXPECT().GetExecutions(gomock.Any(), gomock.Any()).Return(executions, nil)
	mockOutput.EXPECT().ReadLog(gomock.Any(), "exec-1", "workflow-a").
		Return(io.NopCloser(strings.NewReader("log line 1\n")), nil)
	mockOutput.EXPECT().ReadLog(gomock.Any(), "exec-2", "workflow-a").
		Return(io.NopCloser(strings.NewReader("log line 2\n")), nil)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	gr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		content, _ := io.ReadAll(tr)
		files[header.Name] = string(content)
	}

	assert.Contains(t, files, "executions/exec-1.json")
	assert.Contains(t, files, "executions/exec-2.json")
	assert.Contains(t, files, "logs/exec-1.log")
	assert.Contains(t, files, "logs/exec-2.log")
	assert.Contains(t, files, "sequences.json")

	// Verify sequence number tracks the highest
	var seqs []sequenceEntry
	require.NoError(t, json.Unmarshal([]byte(files["sequences.json"]), &seqs))
	assert.Len(t, seqs, 1)
	assert.Equal(t, "workflow-a", seqs[0].WorkflowName)
	assert.Equal(t, int32(5), seqs[0].Number)
}

func TestExportExecutionsHandler_SizeLimitExceeded(t *testing.T) {
	// Use a tiny limit so even a small execution exceeds it.
	// With maxSize=10 the budget is exhausted after writing the execution
	// JSON metadata, so the log read is skipped (capped by remaining budget).
	app, ctrl, mockRepo, _ := setupExportTestServer(t, 10)
	defer ctrl.Finish()

	executions := []testkube.TestWorkflowExecution{
		{
			Id:     "exec-1",
			Number: 1,
			Workflow: &testkube.TestWorkflow{
				Name: "workflow-a",
			},
		},
	}

	mockRepo.EXPECT().GetExecutions(gomock.Any(), gomock.Any()).Return(executions, nil)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestExportExecutionsHandler_SizeLimitFallback(t *testing.T) {
	// exportArchiveMaxSize=0 should fall back to 100 MB
	app, ctrl, mockRepo, _ := setupExportTestServer(t, 0)
	defer ctrl.Finish()

	mockRepo.EXPECT().GetExecutions(gomock.Any(), gomock.Any()).Return(nil, nil)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Empty archive should succeed with the fallback limit
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestExportExecutionsHandler_ValidSinceDate(t *testing.T) {
	app, ctrl, mockRepo, _ := setupExportTestServer(t, 100*1024*1024)
	defer ctrl.Finish()

	mockRepo.EXPECT().GetExecutions(gomock.Any(), gomock.Any()).Return(nil, nil)

	// YYYY-MM-DD format
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export?since=2025-01-01", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestExportExecutionsHandler_ValidSinceDateRFC3339(t *testing.T) {
	app, ctrl, mockRepo, _ := setupExportTestServer(t, 100*1024*1024)
	defer ctrl.Finish()

	mockRepo.EXPECT().GetExecutions(gomock.Any(), gomock.Any()).Return(nil, nil)

	// RFC 3339 format
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export?since=2025-01-01T00:00:00Z", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestExportExecutionsHandler_PostGzipCloseRecheck(t *testing.T) {
	// Set limit that is large enough for mid-write check but will be
	// exceeded after gzip finalization (Close() flushes remaining data).
	// We use a moderate limit and a payload designed to be right at the edge.
	app, ctrl, mockRepo, mockOutput := setupExportTestServer(t, 200)
	defer ctrl.Finish()

	// Create execution data that compresses to roughly the limit
	executions := []testkube.TestWorkflowExecution{
		{
			Id:     "exec-1",
			Number: 1,
			Workflow: &testkube.TestWorkflow{
				Name: "workflow-a",
			},
		},
	}

	mockRepo.EXPECT().GetExecutions(gomock.Any(), gomock.Any()).Return(executions, nil)
	mockOutput.EXPECT().ReadLog(gomock.Any(), "exec-1", "workflow-a").
		Return(io.NopCloser(strings.NewReader(strings.Repeat("x", 300))), nil)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/export", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Should be 413 because the archive exceeds 200 bytes after finalization
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}
