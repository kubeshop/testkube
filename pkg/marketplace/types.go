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
// Field tags are explicit so that the marketplace CLI's JSON and YAML output
// use stable lowercase names regardless of Go field naming.
type Parameter struct {
	Key         string `json:"key" yaml:"key"`
	Default     string `json:"default" yaml:"default"`
	Value       string `json:"value" yaml:"value"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description" yaml:"description"`
	Sensitive   bool   `json:"sensitive" yaml:"sensitive"`
}
