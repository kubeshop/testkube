// Package marketplace provides a client for the Testkube Marketplace
// (https://github.com/kubeshop/testkube-marketplace) catalog. It fetches the
// catalog index, workflow YAML, and readme content, and exposes helpers to
// extract and apply TestWorkflow spec.config parameters.
package marketplace

// Workflow is an entry from the marketplace catalog-index.json file. The
// shape mirrors the CatalogWorkflow type used by the Dashboard UI.
type Workflow struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Component      string `json:"component"`
	ValidationType string `json:"validationType,omitempty"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Icon           string `json:"icon,omitempty"`
	Readme         string `json:"readme,omitempty"`
}

// Parameter describes a single entry from a TestWorkflow's spec.config block.
// Value starts equal to Default and is mutated as the user supplies overrides.
type Parameter struct {
	Key         string
	Default     string
	Value       string
	Type        string
	Description string
	Sensitive   bool
}
