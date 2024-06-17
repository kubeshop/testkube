package testworkflowcontroller

import (
	"bufio"
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ReadTimestamp_UTC(t *testing.T) {
	prefix := "2024-06-07T12:41:49.037275300Z "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + message))
	ts, byt, err := ReadTimestamp(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err)
	assert.Equal(t, []byte(prefix), byt)
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), ts)
}

func Test_ReadTimestamp_NonUTC(t *testing.T) {
	prefix := "2024-06-07T15:41:49.037275300+03:00 "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + message))
	ts, byt, err := ReadTimestamp(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err)
	assert.Equal(t, []byte(prefix), byt)
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), ts)
}
