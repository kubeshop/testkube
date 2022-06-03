package analytics

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendData_SendsToAllSenders(t *testing.T) {
	// given
	var called int32
	payload := NewAPIPayload("cluster", "name", "v1", "host")
	senders := []Sender{
		func(client *http.Client, payload Payload) (out string, err error) {
			atomic.AddInt32(&called, 1)
			return "", nil
		},
		func(client *http.Client, payload Payload) (out string, err error) {
			atomic.AddInt32(&called, 1)
			return "", nil
		},
	}

	// when
	SendData(senders, payload)

	// then
	assert.Equal(t, int32(2), called)
}
