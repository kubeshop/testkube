package instructions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSprintOutputRejectsNonObjects(t *testing.T) {
	assert.Panics(t, func() { SprintOutput("ref", "name", []string{"a"}) })
	assert.Panics(t, func() { SprintOutput("ref", "name", "text") })
	assert.Panics(t, func() { SprintOutput("ref", "name", 1) })
	assert.Panics(t, func() { SprintOutput("ref", "name", map[string]any(nil)) })
	assert.Panics(t, func() { SprintOutput("ref", "name", nil) })
}

func TestSprintOutputAcceptsObjects(t *testing.T) {
	line := SprintOutput("step", "outputs", map[string]string{"token": "abc"})
	instruction := detect(t, line)
	assert.Equal(t, "outputs", instruction.Name)
	assert.Equal(t, map[string]any{"token": "abc"}, instruction.Value)

	line = SprintOutput("step", "status", struct {
		Name string `json:"name"`
	}{Name: "a"})
	instruction = detect(t, line)
	assert.Equal(t, map[string]any{"name": "a"}, instruction.Value)
}

func detect(t *testing.T, line string) *Instruction {
	t.Helper()
	var instruction *Instruction
	for _, part := range splitLines(line) {
		got, _, err := DetectInstruction(part)
		if got == nil {
			continue
		}
		require.NoError(t, err)
		instruction = got
	}
	require.NotNil(t, instruction)
	return instruction
}

func splitLines(s string) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			continue
		}
		lines = append(lines, []byte(s[start:i]))
		start = i + 1
	}
	if start < len(s) {
		lines = append(lines, []byte(s[start:]))
	}
	return lines
}
