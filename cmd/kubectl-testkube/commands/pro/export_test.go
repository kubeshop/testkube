package pro

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestWriteExecutionJSON(t *testing.T) {
	dir := t.TempDir()

	execution := testkube.TestWorkflowExecution{
		Id:     "exec-123",
		Name:   "test-execution",
		Number: 5,
		Workflow: &testkube.TestWorkflow{
			Name: "my-workflow",
		},
	}

	err := writeExecutionJSON(dir, execution)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "exec-123.json"))
	require.NoError(t, err)

	var loaded testkube.TestWorkflowExecution
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, "exec-123", loaded.Id)
	assert.Equal(t, "test-execution", loaded.Name)
	assert.Equal(t, int32(5), loaded.Number)
	assert.Equal(t, "my-workflow", loaded.Workflow.Name)
}

func TestWriteSequences(t *testing.T) {
	dir := t.TempDir()

	sequences := map[string]int32{
		"workflow-a": 10,
		"workflow-b": 25,
	}

	err := writeSequences(dir, sequences)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "sequences.json"))
	require.NoError(t, err)

	var entries []SequenceEntry
	err = json.Unmarshal(data, &entries)
	require.NoError(t, err)

	assert.Len(t, entries, 2)

	seqMap := map[string]int32{}
	for _, e := range entries {
		seqMap[e.WorkflowName] = e.Number
	}
	assert.Equal(t, int32(10), seqMap["workflow-a"])
	assert.Equal(t, int32(25), seqMap["workflow-b"])
}

func TestCreateArchive(t *testing.T) {
	// Create staging directory with test files
	stagingDir := t.TempDir()

	execDir := filepath.Join(stagingDir, "executions")
	logsDir := filepath.Join(stagingDir, "logs")
	require.NoError(t, os.MkdirAll(execDir, 0o755))
	require.NoError(t, os.MkdirAll(logsDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(execDir, "exec-1.json"), []byte(`{"id":"exec-1"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "exec-1.log"), []byte("some log content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stagingDir, "sequences.json"), []byte(`[]`), 0o644))

	outputDir := t.TempDir()

	archivePath, err := createArchive(outputDir, stagingDir)
	require.NoError(t, err)
	assert.FileExists(t, archivePath)

	// Verify archive contents
	f, err := os.Open(archivePath)
	require.NoError(t, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()

	tr := tar.NewReader(gz)

	files := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			files[header.Name] = string(content)
		}
	}

	assert.Contains(t, files, "executions/exec-1.json")
	assert.Contains(t, files, "logs/exec-1.log")
	assert.Contains(t, files, "sequences.json")
	assert.Equal(t, `{"id":"exec-1"}`, files["executions/exec-1.json"])
	assert.Equal(t, "some log content", files["logs/exec-1.log"])
}
