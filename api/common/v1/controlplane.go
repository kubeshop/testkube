package v1

// Control Plane resources (api/insights and the other entity groups) share
// the types below.

// Sources the Control Plane records as creationSource and
// lastModificationSource of a Control Plane resource.
const (
	SourceUI     = "ui"
	SourceAPI    = "api"
	SourceGitOps = "gitops"
)

// ControlPlaneResourceStatus is the Control Plane's view of a resource. The Control Plane
// fills it in when it exports a resource, and ignores it when one is applied:
// ownership is never taken from a request body.
// +kubebuilder:object:generate=true
type ControlPlaneResourceStatus struct {
	// ID is the Control Plane's identifier of the resource.
	ID string `json:"id,omitempty"`
	// GitOpsOwner is the GitOps agent that owns the resource. While it is set,
	// the resource is read-only everywhere else.
	GitOpsOwner *GitOpsOwner `json:"gitOpsOwner,omitempty"`
	// CreationSource records how the resource was created: ui, api or gitops.
	CreationSource string `json:"creationSource,omitempty"`
	// LastModificationSource records how the resource was last changed: ui,
	// api or gitops.
	LastModificationSource string `json:"lastModificationSource,omitempty"`
}

// GitOpsOwner identifies the GitOps agent that owns a resource.
// +kubebuilder:object:generate=true
type GitOpsOwner struct {
	// AgentID is the authoritative identifier of the owning agent.
	AgentID string `json:"agentId"`
	// AgentName is for display only, and may be stale after a rename.
	AgentName string `json:"agentName,omitempty"`
}
