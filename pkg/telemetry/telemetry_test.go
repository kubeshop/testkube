package telemetry

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// The CLI prints this output with --verbose, so that users can inspect what is sent (#3886).
func TestSendData_ReturnsWhatEachSenderSent(t *testing.T) {
	t.Setenv("DO_NOT_TRACK", "")
	senders := map[string]Sender{
		"s2": func(client *http.Client, payload Payload) (out string, err error) {
			return `{"b":2}`, errors.New("unavailable")
		},
		"s1": func(client *http.Client, payload Payload) (out string, err error) {
			return `{"a":1}`, nil
		},
	}

	out, err := sendData(senders, NewAPIPayload("cluster", "name", "v1", "host", "test-cluster", nil))

	assert.NoError(t, err)
	assert.Equal(t, "s1: {\"a\":1}\ns2: {\"b\":2} (error: unavailable)", out)
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

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection reset")
}

// A request can fail after its body is written, so a failure still shows what was sent.
func TestGoogleAnalyticsSender_ReturnsThePayloadOnNetworkErrors(t *testing.T) {
	failing := &http.Client{Transport: failingTransport{}}
	previous := client
	client = failing
	t.Cleanup(func() { client = previous })
	payload := NewAPIPayload("cluster", "testkube_api_heartbeat", "v1", "host", "test-cluster", nil)

	out, err := GoogleAnalyticsSender(failing, payload)

	require.Error(t, err)
	assert.Contains(t, out, `"testkube_api_heartbeat"`)
	assert.Contains(t, out, `"cluster"`)
}
