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

// Funnel joins on email; Segment drops events without a user.
func TestInstallLocalPayload_UserIsTheLicenseOwner(t *testing.T) {
	tests := map[string]struct {
		email    string
		wantUser string
	}{
		"owner known":   {"owner@example.com", "owner@example.com"},
		"lookup failed": {"", GetMachineID()},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			payload := installLocalPayload("v1", "testkube_install_local_attempt", "key", tt.email, map[string]any{"path": "docker"})

			assert.Equal(t, tt.wantUser, payload.UserID)
			assert.Equal(t, "docker", mapProperties(payload.Events[0].Name, payload.Events[0].Params)["path"])
		})
	}
}
