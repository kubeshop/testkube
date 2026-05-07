package testkube

// TestTriggerContentSelector defines what content to watch for trigger events
type TestTriggerContentSelector struct {
	Git *TestTriggerContentGit `json:"git,omitempty"`
}
