package controller

import (
	"bufio"
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ReadTimestamp_UTC_Initial(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T12:41:49.037275300Z "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + message))
	err := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}

func Test_ReadTimestamp_NonUTC_Initial(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T15:41:49.037275300+03:00 "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + message))
	err := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}

func Test_ReadTimestamp_UTC_Recurring(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T12:41:49.037275300Z "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + prefix + message))
	err1 := reader.Read(buf)
	err2 := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}

func Test_ReadTimestamp_NonUTC_Recurring(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T15:41:49.037275300+03:00 "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + prefix + message))
	err1 := reader.Read(buf)
	err2 := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}
