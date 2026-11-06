// Package v1alpha1 contains the API schema of the AI resources: AI Agents,
// the MCP servers they use and the LLM providers that serve their models.
// They live only in the Control Plane and are declared in YAML, to be applied
// by the CLI or, as custom resources, synced by a GitOps agent.
//
// Both paths share this schema and the rules in pkg/controlplane/declarative.
// A manifest never carries a secret: a key is named by an Environment
// Credential in credentialReference, and stays in the Control Plane.
// +kubebuilder:object:generate=true
// +groupName=ai.testkube.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// Group is the API group of the AI resources.
	Group = "ai.testkube.io"

	// Version is the API version of the AI resources.
	Version = "v1alpha1"

	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&AIAgent{},
		&AIAgentList{},
		&MCPServer{},
		&MCPServerList{},
		&LLMProvider{},
		&LLMProviderList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
