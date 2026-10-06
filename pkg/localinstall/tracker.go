package localinstall

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// Client-side project key, public by design like the dashboard's.
	postHogKey      = "phc_zNRKh8Ph9Y6JcSoSNQhZuM7BmQPVyGxiwUYbRaC6VYdA"
	postHogEndpoint = "https://t.testkube.io/capture/"
)

const Notice = "Testkube sends install progress to help us fix setup problems. Opt out: DO_NOT_TRACK=1. Details: docs.testkube.io/articles/telemetry"

type Tracker struct {
	enabled    bool
	distinctID string
	sessionID  string
	version    string
	endpoint   string
	client     *http.Client
	wg         sync.WaitGroup
}

func NewTracker(telemetryEnabled bool, machineID, version string) *Tracker {
	return &Tracker{
		enabled:    telemetryEnabled && !doNotTrack(),
		distinctID: machineID,
		sessionID:  uuid.NewString(),
		version:    version,
		endpoint:   postHogEndpoint,
		client:     &http.Client{Timeout: 2 * time.Second},
	}
}

func (t *Tracker) Enabled() bool {
	return t.enabled
}

// Background send: a slow network must never stall the install.
func (t *Tracker) Send(event string, props map[string]any) {
	if !t.enabled {
		return
	}
	properties := map[string]any{
		"install_session_id": t.sessionID,
		"installer_version":  t.version,
		"os":                 runtime.GOOS,
		"arch":               runtime.GOARCH,
	}
	for k, v := range props {
		properties[k] = v
	}
	body, err := json.Marshal(map[string]any{
		"api_key":     postHogKey,
		"event":       event,
		"distinct_id": t.distinctID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"properties":  properties,
	})
	if err != nil {
		return
	}
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		resp, err := t.client.Post(t.endpoint, "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
		}
	}()
}

// Call before exiting; os.Exit skips deferred calls.
func (t *Tracker) Wait() {
	t.wg.Wait()
}

func doNotTrack() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DO_NOT_TRACK")))
	return v != "" && v != "0" && v != "false"
}
