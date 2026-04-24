/*
Copyright 2021.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package v1 contains API Schema definitions for WorkflowTrigger
// +kubebuilder:object:generate=true
// +groupName=testworkflows.testkube.io
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// Group represents the API Group (shared with TestWorkflow)
	Group = "testworkflows.testkube.io"

	// Version represents the Resource version
	Version = "v1"

	// Kind is the CRD Kind.
	Kind = "WorkflowTrigger"

	// Resource is the plural resource name used for API calls (dynamic client, discovery).
	Resource = "workflowtriggers"

	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// GroupVersionKind identifies these objects by kind.
	GroupVersionKind = schema.GroupVersionKind{Group: Group, Version: Version, Kind: Kind}

	// GroupVersionResource identifies these objects by plural resource name.
	GroupVersionResource = schema.GroupVersionResource{Group: Group, Version: Version, Resource: Resource}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&WorkflowTrigger{},
		&WorkflowTriggerList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
