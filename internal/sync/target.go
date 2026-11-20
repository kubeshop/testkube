package sync

// SourceNamespaceLabel is the label that the agent adds to the copy of a resource that it sends to
// the Control Plane. Its value is the Kubernetes namespace of the resource.
const SourceNamespaceLabel = "testkube.io/source-namespace"

// Target is where a store sends one resource: the Kubernetes namespace that the resource comes
// from, and the Environment that the namespace syncs into.
type Target struct {
	Namespace string
	// EnvironmentID is the Environment of the resource; empty means the Environment of the agent token.
	EnvironmentID string
}

// TargetFunc works out the Target of a resource from its Kubernetes namespace.
type TargetFunc func(namespace string) Target
