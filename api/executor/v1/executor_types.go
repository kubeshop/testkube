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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExecutorSpec defines the desired state of Executor
type ExecutorSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Types defines what types can be handled by executor e.g. "postman/collection", ":curl/command" etc
	Types []string `json:"types,omitempty"`

	// ExecutorType one of "rest" for rest openapi based executors or "job" which will be default runners for testkube
	// or "container" for container executors
	ExecutorType ExecutorType `json:"executor_type,omitempty"`

	// URI for rest based executors
	URI string `json:"uri,omitempty"`

	// Image for kube-job
	Image string `json:"image,omitempty"`
	// executor binary arguments
	Args []string `json:"args,omitempty"`
	// executor default binary command
	Command []string `json:"command,omitempty"`
	// container executor default image pull secrets
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Features list of possible features which executor handles
	Features []Feature `json:"features,omitempty"`

	// ContentTypes list of handled content types
	ContentTypes []ScriptContentType `json:"content_types,omitempty"`

	// Job template to launch executor
	JobTemplate string `json:"job_template,omitempty"`
	// name of the template resource
	JobTemplateReference string `json:"jobTemplateReference,omitempty"`

	// Meta data about executor
	Meta *ExecutorMeta `json:"meta,omitempty"`

	// Slaves data to run test in distributed environment
	Slaves *SlavesMeta `json:"slaves,omitempty"`

	// use data dir as working dir for executor
	UseDataDirAsWorkingDir bool `json:"useDataDirAsWorkingDir,omitempty"`
}

type SlavesMeta struct {
	Image string `json:"image"`
}

// +kubebuilder:validation:Enum=artifacts;junit-report
type Feature string

const (
	FeatureArtifacts   Feature = "artifacts"
	FeatureJUnitReport Feature = "junit-report"
)

// +kubebuilder:validation:Enum=job;container
type ExecutorType string

const (
	ExecutorTypeJob       ExecutorType = "job"
	ExecutorTypeContainer ExecutorType = "container"
)

// +kubebuilder:validation:Enum=string;file-uri;git-file;git-dir;git
type ScriptContentType string

const (
	ScriptContentTypeString  ScriptContentType = "string"
	ScriptContentTypeFileURI ScriptContentType = "file-uri"
	// Deprecated: use git instead
	ScriptContentTypeGitFile ScriptContentType = "git-file"
	// Deprecated: use git instead
	ScriptContentTypeGitDir ScriptContentType = "git-dir"
	ScriptContentTypeGit    ScriptContentType = "git"
)

// Executor meta data
type ExecutorMeta struct {
	// URI for executor icon
	IconURI string `json:"iconURI,omitempty"`
	// URI for executor docs
	DocsURI string `json:"docsURI,omitempty"`
	// executor tooltips
	Tooltips map[string]string `json:"tooltips,omitempty"`
}

type Runner struct {
}

// ExecutorStatus defines the observed state of Executor
type ExecutorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Executor is the Schema for the executors API
type Executor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutorSpec   `json:"spec,omitempty"`
	Status ExecutorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExecutorList contains a list of Executor
type ExecutorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Executor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Executor{}, &ExecutorList{})
}
