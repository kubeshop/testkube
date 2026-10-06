package telemetry

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendData_SendsToAllSenders(t *testing.T) {
	// given
	t.Setenv("DO_NOT_TRACK", "")
	var called int32
	payload := NewAPIPayload("cluster", "name", "v1", "host", "test-cluster", nil)
	senders := map[string]Sender{
		"s1": func(client *http.Client, payload Payload) (out string, err error) {
			atomic.AddInt32(&called, 1)
			return "", nil
		},
		"s2": func(client *http.Client, payload Payload) (out string, err error) {
			atomic.AddInt32(&called, 1)
			return "", nil
		},
	}

	// when
	sendData(senders, payload)

	// then
	assert.Equal(t, int32(2), called)
}

func TestSendData_DoNotTrackSendsNothing(t *testing.T) {
	t.Setenv("DO_NOT_TRACK", "1")
	var called int32
	senders := map[string]Sender{
		"s1": func(client *http.Client, payload Payload) (out string, err error) {
			atomic.AddInt32(&called, 1)
			return "", nil
		},
	}

	sendData(senders, NewAPIPayload("cluster", "name", "v1", "host", "test-cluster", nil))

	assert.Equal(t, int32(0), called)
}
