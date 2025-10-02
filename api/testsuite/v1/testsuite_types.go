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
	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestSuiteSpec defines the desired state of TestSuite
type TestSuiteSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Before steps is list of tests which will be sequentially orchestrated
	Before []TestSuiteStepSpec `json:"before,omitempty"`
	// Steps is list of tests which will be sequentially orchestrated
	Steps []TestSuiteStepSpec `json:"steps,omitempty"`
	// After steps is list of tests which will be sequentially orchestrated
	After []TestSuiteStepSpec `json:"after,omitempty"`

	Repeats     int    `json:"repeats,omitempty"`
	Description string `json:"description,omitempty"`
	// schedule in cron job format for scheduled test execution
	Schedule string `json:"schedule,omitempty"`

	// DEPRECATED execution params passed to executor
	Params map[string]string `json:"params,omitempty"`
	// Variables are new params with secrets attached
	Variables map[string]Variable `json:"variables,omitempty"`
}

type Variable commonv1.Variable

// TestSuiteStepSpec will of particular type will have config for possible step types
type TestSuiteStepSpec struct {
	Type    string                `json:"type,omitempty"`
	Execute *TestSuiteStepExecute `json:"execute,omitempty"`
	Delay   *TestSuiteStepDelay   `json:"delay,omitempty"`
}

// TestSuiteStepType defines different type of test suite steps
type TestSuiteStepType string

const (
	TestSuiteStepTypeExecute TestSuiteStepType = "execute"
	TestSuiteStepTypeDelay   TestSuiteStepType = "delay"
)

// TestSuiteStepExecute defines step to be executed
type TestSuiteStepExecute struct {
	Namespace     string `json:"namespace,omitempty"`
	Name          string `json:"name,omitempty"`
	StopOnFailure bool   `json:"stopOnFailure,omitempty"`
}

// TestSuiteStepDelay contains step delay parameters
type TestSuiteStepDelay struct {
	// Duration in ms
	Duration int32 `json:"duration,omitempty"`
}

// TestSuiteStatus defines the observed state of TestSuite
type TestSuiteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TestSuite is the Schema for the testsuites API
type TestSuite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestSuiteSpec   `json:"spec,omitempty"`
	Status TestSuiteStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TestSuiteList contains a list of TestSuite
type TestSuiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestSuite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestSuite{}, &TestSuiteList{})
}
