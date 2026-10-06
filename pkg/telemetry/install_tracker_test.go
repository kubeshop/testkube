package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoNotTrack_OnlyTruthyValuesOptOut(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"FALSE", false},
		{"1", true},
		{" 1 ", true},
		{"true", true},
	}

	for _, tt := range tests {
		t.Run("value="+tt.value, func(t *testing.T) {
			t.Setenv("DO_NOT_TRACK", tt.value)
			assert.Equal(t, tt.want, DoNotTrack())
		})
	}
}

func TestInstallTracker_DisabledSendsNothing(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
		}))

		tracker := NewInstallTracker(InstallTrackerConfig{Enabled: enabled, MachineID: "machine", Version: "test", Endpoint: server.URL})
		tracker.Send("install_local_started", nil)
		tracker.Wait()
		server.Close()

		want := int32(0)
		if enabled {
			want = 1
		}
		assert.Equal(t, want, requests.Load(), "enabled=%v", enabled)
	}
}

func TestInstallTracker_IdentifyLinksThisRunToEmail(t *testing.T) {
	type event struct {
		Event      string         `json:"event"`
		DistinctID string         `json:"distinct_id"`
		Properties map[string]any `json:"properties"`
	}
	var mu sync.Mutex
	events := map[string]event{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e event
		_ = json.NewDecoder(r.Body).Decode(&e)
		mu.Lock()
		events[e.Event] = e
		mu.Unlock()
	}))
	defer server.Close()

	tracker := NewInstallTracker(InstallTrackerConfig{Enabled: true, MachineID: "machine", Version: "test", Endpoint: server.URL})
	tracker.Identify("")
	tracker.Send("before", nil)
	tracker.Identify("owner@example.com")
	tracker.Send("after", nil)
	tracker.Wait()

	assert.Len(t, events, 3, "empty email must not send an identify")
	before := events["before"]
	assert.Equal(t, before.Properties["install_session_id"], before.DistinctID, "owners share machines, so per run")
	assert.Equal(t, "machine", before.Properties["machine_id"])
	assert.Equal(t, "owner@example.com", events["$identify"].DistinctID)
	assert.Equal(t, before.DistinctID, events["$identify"].Properties["$anon_distinct_id"])
	assert.Equal(t, "owner@example.com", events["after"].DistinctID)
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

			tracker := NewInstallTracker(InstallTrackerConfig{Enabled: true, MachineID: "machine", Version: tt.version, Endpoint: server.URL})
			tracker.Send("install_local_started", nil)
			tracker.Wait()

			assert.Equal(t, tt.wantDev, payload.Properties["dev_build"])
		})
	}
}
