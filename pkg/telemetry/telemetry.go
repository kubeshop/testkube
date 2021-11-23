package telemetry

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/segmentio/analytics-go"
)

var telemetryToken = ""

const heartbeatEvent = "testkube-heartbeat"

func CollectAnonymousInfo() {

	var isDisabled bool

	if val, ok := os.LookupEnv("TESTKUBE_TELEMETRY_DISABLED"); ok {
		isDisabled, _ = strconv.ParseBool(val)
	}

	if !isDisabled {
		client := analytics.New(telemetryToken)
		client.Enqueue(analytics.Track{
			AnonymousId: machineID(),
			Event:       heartbeatEvent,
			Timestamp:   time.Now(),
		})

		client.Close()
	}
}

func CollectAnonymousCmdInfo(command string) {

	var isDisabled bool

	if val, ok := os.LookupEnv("TESTKUBE_TELEMETRY_DISABLED"); ok {
		isDisabled, _ = strconv.ParseBool(val)
	}
	if !isDisabled {
		client := analytics.New(telemetryToken)
		client.Enqueue(analytics.Track{
			AnonymousId: machineID(),
			Event:       filepath.Join(heartbeatEvent, command),
			Timestamp:   time.Now(),
		})

		client.Close()
	}
}

func machineID() string {
	id, _ := generate()
	return id
}

// Generate returns protected id for the current machine
func generate() (string, error) {
	id, err := machineid.ProtectedID("testkube")
	if err != nil {
		return fromHostname()
	}
	return id, err
}

// fromHostname generates a machine id hash from hostname
func fromHostname() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}
	sum := md5.Sum([]byte(name))
	return hex.EncodeToString(sum[:]), nil
}
