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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestSpec defines the desired state of Test
type TestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Before steps is list of scripts which will be sequentially orchestrated
	Before []TestStepSpec `json:"before,omitempty"`
	// Steps is list of scripts which will be sequentially orchestrated
	Steps []TestStepSpec `json:"steps,omitempty"`
	// After steps is list of scripts which will be sequentially orchestrated
	After []TestStepSpec `json:"after,omitempty"`

	Repeats     int      `json:"repeats,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// TestStepSpec will of particular type will have config for possible step types
type TestStepSpec struct {
	Type    string           `json:"type,omitempty"`
	Execute *TestStepExecute `json:"execute,omitempty"`
	Delay   *TestStepDelay   `json:"delay,omitempty"`
}

type TestStepType string

const (
	TestStepTypeExecute TestStepType = "execute"
	TestStepTypeDelay   TestStepType = "delay"
)

type TestStepExecute struct {
	Namespace     string `json:"namespace,omitempty"`
	Name          string `json:"name,omitempty"`
	StopOnFailure bool   `json:"stopOnFailure,omitempty"`
}

type TestStepDelay struct {
	// Duration in ms
	Duration int32 `json:"duration,omitempty"`
}

// TestStatus defines the observed state of Test
type TestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Test is the Schema for the tests API
type Test struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestSpec   `json:"spec,omitempty"`
	Status TestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TestList contains a list of Test
type TestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Test `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Test{}, &TestList{})
}
