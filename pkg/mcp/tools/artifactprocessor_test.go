package tools

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d content", i+1)
	}
	return strings.Join(lines, "\n")
}

func TestProcessArtifact_DefaultFirst100Lines(t *testing.T) {
	content := []byte(makeLines(500))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{})

	assert.Contains(t, result, "Total lines: 500")
	assert.Contains(t, result, "Showing: lines 1-100 of 500")
	assert.Contains(t, result, "line 1 content")
	assert.Contains(t, result, "line 100 content")
	assert.NotContains(t, result, "line 101 content")
}

func TestProcessArtifact_ShortFile(t *testing.T) {
	content := []byte(makeLines(50))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{})

	assert.Contains(t, result, "Total lines: 50")
	assert.Contains(t, result, "Showing: lines 1-50 of 50")
	assert.Contains(t, result, "line 50 content")
}

func TestProcessArtifact_EmptyArtifact(t *testing.T) {
	result := ProcessArtifact([]byte(""), "test.log", ArtifactReadParams{})
	assert.Contains(t, result, "Total lines: 1")
}

func TestProcessArtifact_LineRange(t *testing.T) {
	content := []byte(makeLines(500))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{StartLine: 200, EndLine: 250})

	assert.Contains(t, result, "Showing: lines 200-250 of 500")
	assert.Contains(t, result, "line 200 content")
	assert.Contains(t, result, "line 250 content")
	assert.NotContains(t, result, "line 199 content")
	assert.NotContains(t, result, "line 251 content")
}

func TestProcessArtifact_LineRangeExceedsMaxLines(t *testing.T) {
	content := []byte(makeLines(500))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{StartLine: 1, EndLine: 400})

	assert.Contains(t, result, "Showing: lines 1-200 of 500")
	assert.Contains(t, result, "line 200 content")
	assert.NotContains(t, result, "line 201 content")
}

func TestProcessArtifact_StartLineOnly(t *testing.T) {
	content := []byte(makeLines(500))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{StartLine: 400})

	assert.Contains(t, result, "Showing: lines 400-499 of 500")
	assert.Contains(t, result, "line 400 content")
	assert.Contains(t, result, "line 499 content")
}

func TestProcessArtifact_StartLineOnlyShortFile(t *testing.T) {
	content := []byte(makeLines(450))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{StartLine: 400})

	assert.Contains(t, result, "Showing: lines 400-450 of 450")
	assert.Contains(t, result, "line 400 content")
	assert.Contains(t, result, "line 450 content")
}

func TestProcessArtifact_StartLineBeyondEnd(t *testing.T) {
	content := []byte(makeLines(100))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{StartLine: 200})

	assert.Contains(t, result, "0 lines")
	assert.Contains(t, result, "exceeds total")
	assert.NotContains(t, result, "line 1 content")
}

func TestProcessArtifact_Grep(t *testing.T) {
	lines := []string{
		"line 1 normal",
		"line 2 normal",
		"line 3 ERROR something failed",
		"line 4 normal",
		"line 5 normal",
		"line 6 normal",
		"line 7 ERROR another failure",
		"line 8 normal",
	}
	content := []byte(strings.Join(lines, "\n"))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{Grep: "ERROR"})

	assert.Contains(t, result, "2 grep matches")
	assert.Contains(t, result, "ERROR something failed")
	assert.Contains(t, result, "ERROR another failure")
}

func TestProcessArtifact_GrepCaseInsensitive(t *testing.T) {
	content := []byte("line 1\nERROR here\nerror there\nLine 4\n")
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{Grep: "error"})

	assert.Contains(t, result, "2 grep matches")
}

func TestProcessArtifact_GrepNoMatches(t *testing.T) {
	content := []byte(makeLines(100))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{Grep: "NONEXISTENT"})

	assert.Contains(t, result, "0 grep matches")
}

func TestProcessArtifact_Binary(t *testing.T) {
	content := []byte("PNG\x89\x00\x00binary\x00data")
	result := ProcessArtifact(content, "image.png", ArtifactReadParams{})

	assert.Contains(t, result, "Binary artifact")
	assert.Contains(t, result, "line-based display not available")
}

func TestProcessArtifact_HardCap(t *testing.T) {
	content := []byte(makeLines(1000))
	result := ProcessArtifact(content, "test.log", ArtifactReadParams{StartLine: 1, EndLine: 1000})

	// Should cap at 200 lines
	assert.Contains(t, result, "Showing: lines 1-200 of 1000")
	assert.NotContains(t, result, "line 201 content")
}
