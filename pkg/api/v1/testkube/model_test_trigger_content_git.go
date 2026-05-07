package testkube

// TestTriggerContentGit defines git repository configuration for content triggers
type TestTriggerContentGit struct {
	// URI of the git repository
	Uri string `json:"uri"`
	// Revision (branch name, tag, or commit SHA) to watch
	Revision string `json:"revision,omitempty"`
	// Paths is a list of file/directory paths to watch for changes
	Paths []string `json:"paths,omitempty"`
}
