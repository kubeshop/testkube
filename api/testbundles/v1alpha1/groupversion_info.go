// Package v1alpha1 contains the API schema of Test Bundles: the definition of
// a bundle and the guidance its authoring uses. Test Bundles live only in the
// Control Plane and are declared in YAML, to be applied by the CLI or, as
// custom resources, synced by a GitOps agent. What a bundle's authoring
// produces (test files, chats, its sandbox and shared workflow) stays in the
// Control Plane and is not declared.
//
// Both paths share this schema and the rules in pkg/controlplane/declarative.
// +kubebuilder:object:generate=true
// +groupName=testbundles.testkube.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// Group is the API group of Test Bundles.
	Group = "testbundles.testkube.io"

	// Version is the API version of Test Bundles.
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
		&TestBundle{},
		&TestBundleList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
