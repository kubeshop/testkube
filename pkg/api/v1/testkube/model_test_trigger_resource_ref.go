package testkube

// TestTriggerResourceRef identifies a K8s resource by Group/Version/Kind.
type TestTriggerResourceRef struct {
	// API group (empty for core resources like Pod, Service)
	Group string `json:"group,omitempty"`
	// API version
	Version string `json:"version,omitempty"`
	// Resource kind (e.g. Deployment, KafkaTopic)
	Kind string `json:"kind"`
}
