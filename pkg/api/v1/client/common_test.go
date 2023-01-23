package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/executor/output"
)

func TestTrimSSEChunk(t *testing.T) {

	in := []byte(`data: {"type": "error","message": "some message"}\n\n`)
	out := trimDataChunk(in)

	assert.Equal(t, `{"type": "error","message": "some message"}`, string(out))
}

// TestStreamToLogsChannelOldErrorFormat parses old output error format and return type field
func TestStreamToLogsChannelOldErrorFormat(t *testing.T) {
	log := make(chan output.Output)
	in := []byte(`data: {"type": "error", "message": "some message"}` + "\n\n")
	buf := bytes.NewBuffer(in)

	go StreamToLogsChannel(buf, log)
	result := <-log
	assert.Equal(t, output.Output{Type_: "error", Content: ""}, result)
}

// TestStreamToLogsChannelNewErrorFormat parses new output error format and return type and content fields
func TestStreamToLogsChannelNewErrorFormat(t *testing.T) {
	log := make(chan output.Output)
	out, _ := json.Marshal(output.NewOutputError(errors.New("some message")))
	in := []byte(fmt.Sprintf("%s\n", out))
	buf := bytes.NewBuffer(in)

	go StreamToLogsChannel(buf, log)
	result := <-log
	assert.Equal(t, "error", result.Type_)
	assert.Equal(t, "some message", result.Content)
}
