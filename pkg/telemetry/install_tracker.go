package telemetry

import (
	"bytes"
	"context"
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

const InstallNotice = "Testkube sends install progress to help us fix setup problems. Opt out: DO_NOT_TRACK=1. Details: docs.testkube.io/articles/telemetry"

// InstallTracker reports `testkube install local` steps to PostHog
// project "On Prem Trials" through https://t.testkube.io/capture/,
// keyed by the CLI machine ID plus a per-run install_session_id.
// Sends run in the background with a 2s timeout.
// DO_NOT_TRACK or telemetryEnabled false sends nothing, hides the notice.
// Events: install_local_started, install_local_checks (one per run),
// install_local_failed, install_local_aborted, install_local_checks_done.
type InstallTracker struct {
	enabled    bool
	distinctID string
	sessionID  string
	version    string
	endpoint   string
	client     *http.Client
	wg         sync.WaitGroup
}

type InstallTrackerConfig struct {
	Enabled   bool
	MachineID string
	Version   string
	// Empty means production; tests point it at a local server.
	Endpoint string
}

func NewInstallTracker(cfg InstallTrackerConfig) *InstallTracker {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = postHogEndpoint
	}
	return &InstallTracker{
		enabled:    cfg.Enabled,
		distinctID: cfg.MachineID,
		sessionID:  uuid.NewString(),
		version:    cfg.Version,
		endpoint:   endpoint,
		client:     &http.Client{Timeout: 2 * time.Second},
	}
}

func (t *InstallTracker) Enabled() bool {
	return t.enabled
}

// Background send: a slow network must never stall the install.
func (t *InstallTracker) Send(event string, props map[string]any) {
	if !t.enabled {
		return
	}
	properties := map[string]any{
		"install_session_id": t.sessionID,
		"installer_version":  t.version,
		"os":                 runtime.GOOS,
		"arch":               runtime.GOARCH,
		// Local builds report 999.0.0-*; lets PostHog filter our own runs.
		"dev_build": strings.HasPrefix(t.version, "999.0.0"),
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
		// Not the command context: aborted events must send after Ctrl+C.
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, t.endpoint, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := t.client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
}

// Call before exiting; os.Exit skips deferred calls.
func (t *InstallTracker) Wait() {
	t.wg.Wait()
}

// DO_NOT_TRACK is the cross-tool opt-out convention.
func DoNotTrack() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DO_NOT_TRACK")))
	return v != "" && v != "0" && v != "false"
}
