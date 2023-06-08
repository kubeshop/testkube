package telemetry

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMapParams(t *testing.T) {
	// given
	payload := Payload{
		UserID:   "u1",
		ClientID: "c1",
		Events: []Event{
			{
				Name: "e1",
				Params: Params{
					AppName:         "testkube-api",
					AppVersion:      "v1.0.0",
					Architecture:    "amd64",
					OperatingSystem: "linux",
					MachineID:       "mid1",
					ClusterID:       "cid1",
					EventCategory:   "command",
					Host:            "local",
					DataSource:      "git-dir",
					TestType:        "postman/collection",
					DurationMs:      100,
					Status:          "failed",
					TestSource:      "main",
					TestSuiteSteps:  5,
					ClusterType:     "local",
				},
			},
		},
	}

	// when
	track := mapEvent(payload.UserID, payload.Events[0])

	// then
	assert.Equal(t, "testkube-api", track.Properties["name"])
	assert.Equal(t, "v1.0.0", track.Properties["version"])
	assert.Equal(t, "amd64", track.Properties["arch"])
	assert.Equal(t, "linux", track.Properties["os"])
	assert.Equal(t, "cid1", track.Properties["clusterId"])
	assert.Equal(t, "mid1", track.Properties["machineId"])
	assert.Equal(t, "command", track.Properties["eventCategory"])
	assert.Equal(t, "local", track.Properties["host"])
	assert.Equal(t, "git-dir", track.Properties["dataSource"])
	assert.Equal(t, "postman/collection", track.Properties["testType"])
	assert.Equal(t, int32(100), track.Properties["durationMs"])
	assert.Equal(t, "failed", track.Properties["status"])
	assert.Equal(t, "main", track.Properties["testSource"])
	assert.Equal(t, int32(5), track.Properties["testSuiteSteps"])
	assert.Equal(t, "local", track.Properties["clusterType"])
}

func TestSegmentioSender(t *testing.T) {
	t.Skip("for debug only, to check if real events are getting into Segment.io")

	// given
	payload := Payload{
		UserID:   "u1",
		ClientID: "c1",
		Events: []Event{
			{
				Name: "kubectl testkube run test",
				Params: Params{
					AppName:         "testkube-api",
					AppVersion:      "v1.0.0",
					Architecture:    "amd64",
					OperatingSystem: "linux",
					MachineID:       "mid1",
					ClusterID:       "cid1",
					EventCategory:   "command",
					Host:            "local",
					DataSource:      "git-dir",
					TestType:        "postman/collection",
					DurationMs:      100,
					Status:          "failed",
					TestSource:      "main",
					TestSuiteSteps:  5,
					ClusterType:     "local",
				},
			},
		},
	}

	// when
	out, err := SegmentioSender(http.DefaultClient, payload)

	time.Sleep(100 * time.Millisecond)

	// then
	assert.NoError(t, err)
	assert.Equal(t, "", out)
	t.Fail()

}
