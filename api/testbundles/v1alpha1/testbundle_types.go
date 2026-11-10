package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

const (
	// TestBundleKind is the kind of a Test Bundle.
	TestBundleKind = "TestBundle"
	// TestBundleResource is the plural name of TestBundleKind.
	TestBundleResource = "testbundles"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TestBundle is a Test Bundle of an environment. metadata.name is its slug,
// which never changes once the bundle exists.
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Workflow",type=string,JSONPath=`.status.workflowName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type TestBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestBundleSpec   `json:"spec,omitempty"`
	Status TestBundleStatus `json:"status,omitempty"`
}

// TestBundleSpec is what a Test Bundle is and the guidance its authoring
// uses.
type TestBundleSpec struct {
	// DisplayName is the bundle's name in the dashboard. It defaults to
	// metadata.name.
	DisplayName string `json:"displayName,omitempty"`
	// Description of the bundle.
	Description string `json:"description,omitempty"`
	// Labels categorize the bundle in the dashboard.
	Labels map[string]string `json:"labels,omitempty"`
	// GitSource is the repository the bundle's tests come from. It is only
	// read from; nothing is pushed back.
	GitSource *TestBundleGitSource `json:"gitSource,omitempty"`
	// Skills are the guidance every authoring chat of the bundle gets, in
	// this order. They belong to the bundle: applying a list replaces the
	// whole list, so a Skill left out is deleted, and an empty list removes
	// them all. When skills is omitted, the bundle's Skills are left as they
	// are, so applying a manifest without them does not delete the ones
	// added in the dashboard.
	// +optional
	Skills []TestBundleSkill `json:"skills"`
	// Contexts are the bundle's Attached Context, in this order: guidance
	// its authoring sandbox holds for the AI to read, never part of the
	// tests the bundle publishes. As with skills, applying a list replaces
	// the whole list, an empty list removes them all, and when contexts is
	// omitted the bundle's Attached Context is left as it is.
	// +optional
	Contexts []TestBundleContext `json:"contexts"`
	// AIAgents are the AI Agents, by metadata.name, the bundle's authoring
	// may delegate to. Each must be public. When it is omitted, a new bundle
	// starts with the environment's defaults for new bundles and an existing
	// one keeps its own; an empty list delegates to none.
	AIAgents *[]string `json:"aiAgents,omitempty"`
}

// TestBundleGitSource is a repository a Test Bundle reads tests from.
type TestBundleGitSource struct {
	// URI of the repository. It must not carry credentials.
	// +kubebuilder:validation:MinLength=1
	URI string `json:"uri"`
	// Revision is a branch, tag or commit.
	Revision string `json:"revision,omitempty"`
	// Path within the repository.
	Path string `json:"path,omitempty"`
}

// TestBundleSkill is guidance a Test Bundle's authoring gets in full in every
// prompt, following the Agent Skills spec.
type TestBundleSkill struct {
	// Name identifies the Skill within the bundle: up to 64 lowercase
	// letters, digits and single hyphens.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[a-z0-9]+(-[a-z0-9]+)*$`
	Name string `json:"name"`
	// Description says what the Skill does and when to use it.
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`
	// Content is the guidance itself.
	// +kubebuilder:validation:MinLength=1
	Content string `json:"content"`
	// Metadata is extra Agent Skills spec metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TestBundleContext is one piece of a Test Bundle's Attached Context.
type TestBundleContext struct {
	// Title names the context in the dashboard. Titles need not be unique.
	// +kubebuilder:validation:MinLength=1
	Title string `json:"title"`
	// Content is the guidance itself.
	// +kubebuilder:validation:MinLength=1
	Content string `json:"content"`
}

// TestBundleStatus is the Control Plane's view of a Test Bundle. It is
// exported, never applied.
type TestBundleStatus struct {
	commonv1.ControlPlaneResourceStatus `json:",inline"`
	// WorkflowName is the bundle's shared Test Workflow, set once a chat
	// creates it.
	WorkflowName string `json:"workflowName,omitempty"`
	// LastActiveAt is when a chat on the bundle was last active.
	LastActiveAt *metav1.Time `json:"lastActiveAt,omitempty"`
	// SessionCount is the number of chats on the bundle the exporter sees.
	SessionCount int `json:"sessionCount,omitempty"`
}

//+kubebuilder:object:root=true

// TestBundleList contains a list of TestBundle.
type TestBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestBundle `json:"items"`
}
