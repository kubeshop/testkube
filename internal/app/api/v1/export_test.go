package v1

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
