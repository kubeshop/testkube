package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimSSEChunk(t *testing.T) {

	in := []byte(`data: {"type": "error","message": "some message"}\n\n`)
	out := trimDataChunk(in)

	assert.Equal(t, `{"type": "error","message": "some message"}`, string(out))
}
