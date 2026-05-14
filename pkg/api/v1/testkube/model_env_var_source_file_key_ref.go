package testkube

// EnvVarSourceFileKeyRef selects a key from an env file mounted in a volume.
type EnvVarSourceFileKeyRef struct {
	VolumeName string `json:"volumeName"`
	Path       string `json:"path"`
	Key        string `json:"key"`
	Optional   *bool  `json:"optional,omitempty"`
}
