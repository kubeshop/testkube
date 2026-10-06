package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallTracker_OptOutSendsNothing(t *testing.T) {
	tests := []struct {
		name             string
		telemetryEnabled bool
		doNotTrack       string
		wantRequests     int32
	}{
		{"enabled", true, "", 1},
		{"DO_NOT_TRACK=1", true, "1", 0},
		{"DO_NOT_TRACK=true", true, "true", 0},
		{"DO_NOT_TRACK=0 still tracks", true, "0", 1},
		{"telemetry disabled in CLI config", false, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
			}))
			defer server.Close()
			t.Setenv("DO_NOT_TRACK", tt.doNotTrack)

			tracker := NewInstallTracker(tt.telemetryEnabled, "machine", "test")
			tracker.endpoint = server.URL
			tracker.Send("install_local_started", nil)
			tracker.Wait()

			assert.Equal(t, tt.wantRequests, requests.Load())
		})
	}
}

func TestInstallTracker_TagsOnlyLocalBuildsAsDev(t *testing.T) {
	tests := []struct {
		version string
		wantDev bool
	}{
		{"999.0.0-dev", true},
		{"999.0.0-abc1234", true},
		{"2.14.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			var payload struct {
				Properties map[string]any `json:"properties"`
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&payload)
			}))
			defer server.Close()
			t.Setenv("DO_NOT_TRACK", "")

			tracker := NewInstallTracker(true, "machine", tt.version)
			tracker.endpoint = server.URL
			tracker.Send("install_local_started", nil)
			tracker.Wait()

			assert.Equal(t, tt.wantDev, payload.Properties["dev_build"])
		})
	}
}
