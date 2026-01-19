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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestWorkflowSpec defines the desired state of TestWorkflow
type TestWorkflowSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// templates to include at a top-level of workflow
	Use []TemplateRef `json:"use,omitempty" expr:"include"`

	TestWorkflowSpecBase `json:",inline" expr:"include"`

	// list of accompanying services to start
	Services map[string]ServiceSpec `json:"services,omitempty" expr:"template,include"`

	// steps for setting up the workflow
	Setup []Step `json:"setup,omitempty" expr:"include"`

	// steps to execute in the workflow
	Steps []Step `json:"steps,omitempty" expr:"include"`

	// steps to run at the end of the workflow
	After []Step `json:"after,omitempty" expr:"include"`

	// list of accompanying permanent volume claims
	Pvcs map[string]corev1.PersistentVolumeClaimSpec `json:"pvcs,omitempty" expr:"template,include"`
}

// TemplateRef is the reference for the template inclusion
type TemplateRef struct {
	// name of the template to include
	Name string `json:"name"`
	// trait configuration values if needed
	Config map[string]intstr.IntOrString `json:"config,omitempty" expr:"template"`
}

// TestWorkflowStatusSummary contains information about the TestWorkflow status.
// It includes details about the latest execution and health of the TestWorkflow.
type TestWorkflowStatusSummary struct {
	// (Deprecated) LatestExecution maintained for backward compatibility, never actively used
	LatestExecution *TestWorkflowExecutionSummary `json:"latestExecution,omitempty"`
	Health          *TestWorkflowExecutionHealth  `json:"health,omitempty"`
}

// TestWorkflowExecutionHealth provides health information about a test workflow
type TestWorkflowExecutionHealth struct {
	// Recency-weighted fraction of executions that passed (value between 0.0 and 1.0).
	PassRate float64 `json:"passRate"`
	// Fraction of status changes among consecutive executions without recency weighting (value between 0.0 and 1.0).
	FlipRate float64 `json:"flipRate"`
	// Combined health score, computed as passRate * (1 - flipRate) (value between 0.0 and 1.0).
	OverallHealth float64 `json:"overallHealth"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// TestWorkflow is the Schema for the workflows API
type TestWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// TestWorkflow readable description
	Description string `json:"description,omitempty"`

	// TestWorkflow specification
	Spec   TestWorkflowSpec          `json:"spec" expr:"include"`
	Status TestWorkflowStatusSummary `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TestWorkflowList contains a list of TestWorkflow
type TestWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestWorkflow `json:"items" expr:"include"`
}

func init() {
	SchemeBuilder.Register(&TestWorkflow{}, &TestWorkflowList{})
}
