package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBufferedFileWriter(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	meta1 := &Metadata{
		Workflow:  "wf",
		Step:      Step{Ref: "step"},
		Execution: "exec",
		Format:    "txt",
	}
	meta2 := &Metadata{
		Workflow:  "wf2",
		Step:      Step{Ref: "step2", Parent: "step0"},
		Execution: "exec2",
		Format:    "txt",
	}

	writer1, err := NewFileWriter(tmpDir, meta1, 1)
	require.NoError(t, err)
	require.NotNil(t, writer1)
	writer2, err := NewFileWriter(tmpDir, meta2, 1)
	require.NoError(t, err)
	require.NotNil(t, writer2)

	// Ensure the correct file was created
	expectedFilename1 := fmt.Sprintf("%s_%s_%s.%s", meta1.Workflow, meta1.Step.Ref, meta1.Execution, meta1.Format)
	fullPath := filepath.Join(tmpDir, expectedFilename1)
	_, statErr := os.Stat(fullPath)
	assert.NoError(t, statErr, "expected the file to exist at %s", fullPath)
	expectedFilename2 := fmt.Sprintf("%s_%s_%s_%s.%s", meta2.Workflow, meta2.Step.Ref, meta2.Execution, meta2.Step.Parent, meta1.Format)
	fullPath = filepath.Join(tmpDir, expectedFilename2)
	_, statErr = os.Stat(fullPath)
	assert.NoError(t, statErr, "expected the file to exist at %s", fullPath)

	// Cleanup
	require.NoError(t, writer1.Close(context.Background()))
	require.NoError(t, writer2.Close(context.Background()))
}

func TestInitFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := "test_metrics.txt"
	fullPath := filepath.Join(tmpDir, filename)
	t.Cleanup(func() {
		_ = os.Remove(fullPath)
	})

	f, err := initFile(tmpDir, filename)
	require.NoError(t, err, "initFile should create file successfully")
	require.NotNil(t, f, "returned file should not be nil")
	t.Cleanup(func() {
		_ = f.Close()
	})

	// Check that file has reserved metadata space
	info, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, info.Size(), int64(headerEndIndex+1), "file should have reserved at least 256 bytes")

	// Validate the reserved bytes in the file
	content := make([]byte, headerEndIndex+1)
	_, err = f.ReadAt(content, 0)
	require.NoError(t, err)

	// Last character should be a newline; rest should be 0x00
	assert.Equal(t, byte('\n'), content[headerEndIndex], "last reserved byte should be newline")
	for i := 0; i < headerEndIndex; i++ {
		assert.Equal(t, byte(0x00), content[i], "file metadata space should be null bytes")
	}
}

func TestReserveMetadataSpace(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "reserve_test.txt")

	// Create file manually
	f, err := os.Create(filePath)
	require.NoError(t, err)
	defer f.Close()

	// Call the unexported function directly (same-package test).
	err = reserveMetadataSpace(f)
	require.NoError(t, err, "expected no error reserving metadata space")

	// Validate the reserved space
	content := make([]byte, headerLength)
	_, err = f.ReadAt(content, 0)
	require.NoError(t, err)

	assert.Equal(t, byte('\n'), content[headerEndIndex], "last reserved byte should be newline")
	for i := 0; i < headerEndIndex-1; i++ {
		assert.Equal(t, byte(0x00), content[i], "all other reserved bytes should be null")
	}
}

func TestBufferedFileWriter_Write(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	meta := &Metadata{
		Workflow:  "wf",
		Step:      Step{Ref: "step"},
		Execution: "exec",
		Format:    "txt",
		Lines:     0,
	}

	ctx := context.Background()

	writer, err := NewFileWriter(tmpDir, meta, 1)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = writer.Close(context.Background())
	})

	// 1) Write some data
	err = writer.Write(ctx, "Hello, world!")
	require.NoError(t, err)

	// 2) Check if lines increment
	assert.Equal(t, 1, meta.Lines, "metadata Lines should increment to 1")

	// 3) Write some more data
	require.NoError(t, writer.Write(ctx, "Another line"))
	assert.Equal(t, 2, meta.Lines, "metadata Lines should increment to 2")

	// 4) Close the writer and assert that it cannot be written to anymore.
	assert.NoError(t, writer.Close(ctx))
	assert.Error(t, writer.Write(ctx, "Should fail"))
}

func TestBufferedFileWriter_writeMetadata(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	meta := &Metadata{
		Workflow:  "wf",
		Step:      Step{Ref: "step"},
		Execution: "exec",
		Format:    "influx",
		Lines:     42,
	}
	writer, err := NewFileWriter(tmpDir, meta, 1)
	require.NoError(t, err)
	defer writer.Close(context.Background())

	// 1) Write metadata to the file
	err = writer.writeMetadata(context.Background(), meta)
	require.NoError(t, err)

	// 2) Assert that the file size is equal to the metadata length
	info, err := writer.f.Stat()
	require.NoError(t, err)
	assert.Equal(t, info.Size(), int64(headerEndIndex+1))

	// Read back the metadata length byte and metadata
	controlBuf := make([]byte, 1)
	_, err = writer.f.ReadAt(controlBuf, metadataControlByteIndex)
	require.NoError(t, err)

	metadataLen := len(meta.String())
	metadataBuf := make([]byte, metadataLen)
	_, err = writer.f.ReadAt(metadataBuf, metadataStartIndex)
	require.NoError(t, err)

	// Compare with meta.String()
	expectedMetadata := meta.String()
	assert.Equal(t, len(expectedMetadata), metadataLen, "length byte should match metadata string length")
	assert.Equal(t, expectedMetadata, string(metadataBuf), "metadata written should match meta.String()")
}

func TestBufferedFileWriter_Close(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	meta := &Metadata{
		Workflow:  "wf",
		Step:      Step{Ref: "step"},
		Execution: "exec",
		Format:    "txt",
	}
	writer, err := NewFileWriter(tmpDir, meta, 1)
	require.NoError(t, err)

	err = writer.Write(context.Background(), "line1")
	require.NoError(t, err)
	assert.Equal(t, 1, meta.Lines)

	err = writer.Close(context.Background())
	require.NoError(t, err, "expected no error closing the writer")

	// Verify that:
	//  1) File is closed (subsequent writes should fail).
	//  2) Metadata is written (lines = 1).
	//  3) Buffer is flushed (the data should appear in file).

	// Attempt to write again -> error
	require.Error(t, writer.Write(context.Background(), "should fail"))

	// Validate the file content on disk
	filename := fmt.Sprintf("%s_%s_%s.%s", meta.Workflow, meta.Step.Ref, meta.Execution, meta.Format)
	fullPath := filepath.Join(tmpDir, filename)

	content, readErr := os.ReadFile(fullPath)
	require.NoError(t, readErr)

	// The data "line1" should be there
	assert.Contains(t, string(content), "line1\n")
}
