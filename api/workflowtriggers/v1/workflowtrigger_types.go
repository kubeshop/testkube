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

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WorkflowTrigger is the Schema for the workflowtriggers API
// +kubebuilder:printcolumn:name="Event",type=string,JSONPath=`.spec.when.event`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type WorkflowTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowTriggerSpec   `json:"spec,omitempty"`
	Status WorkflowTriggerStatus `json:"status,omitempty"`
}

// WorkflowTriggerSpec defines the desired state of WorkflowTrigger
type WorkflowTriggerSpec struct {
	// Watch defines which K8s resource to observe. Optional for non-K8s trigger sources.
	Watch *WorkflowTriggerWatch `json:"watch,omitempty"`
	// When defines the trigger source.
	When WorkflowTriggerWhen `json:"when"`
	// Match defines field-level conditions using JSONPath. All conditions must pass (AND logic).
	Match []WorkflowTriggerFieldCondition `json:"match,omitempty"`
	// Wait defines stability gates that must pass before execution.
	Wait *WorkflowTriggerWait `json:"wait,omitempty"`
	// Run defines what workflow to execute and how.
	Run WorkflowTriggerRun `json:"run"`
	// Disabled disables the trigger without deleting it.
	Disabled bool `json:"disabled,omitempty"`
}

// WorkflowTriggerWatch defines what K8s resource to observe.
type WorkflowTriggerWatch struct {
	// Resource identifies the K8s resource by Group/Version/Kind.
	// Group and Version are optional for well-known types (resolved via discovery API).
	Resource WorkflowTriggerResource `json:"resource"`
	// Selector provides advanced resource filtering (regex, labels).
	Selector *WorkflowTriggerSelector `json:"selector,omitempty"`
}

// WorkflowTriggerResource identifies a K8s resource by GVK with optional name/namespace.
type WorkflowTriggerResource struct {
	// Group is the API group (empty for core resources like Pod, Service, ConfigMap).
	Group string `json:"group,omitempty"`
	// Version is the API version. Optional for well-known types.
	Version string `json:"version,omitempty"`
	// Kind is the resource kind (e.g. Deployment, KafkaTopic). Required.
	Kind string `json:"kind"`
	// Name matches the resource by exact name.
	Name string `json:"name,omitempty"`
	// Namespace matches the resource by exact namespace.
	Namespace string `json:"namespace,omitempty"`
}

// WorkflowTriggerSelector provides advanced resource selection criteria.
type WorkflowTriggerSelector struct {
	// NameRegex matches the resource name by regex.
	NameRegex string `json:"nameRegex,omitempty"`
	// NamespaceRegex matches the resource namespace by regex.
	NamespaceRegex string `json:"namespaceRegex,omitempty"`
	// LabelSelector matches resources by K8s label selector.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// WorkflowTriggerWhen defines the trigger source. Structured as a separate type
// to allow future trigger sources (schedule, webhook, git) alongside event.
type WorkflowTriggerWhen struct {
	// Event is the K8s resource event type. Required when no other trigger source
	// (schedule, webhook, etc.) is configured. Validated at application level.
	// +kubebuilder:validation:Enum=created;modified;deleted
	Event string `json:"event,omitempty"`
}

// WorkflowTriggerFieldCondition defines a field-level match condition.
type WorkflowTriggerFieldCondition struct {
	// Path is a dot-path to a field on the K8s object, e.g. ".spec.replicas",
	// ".spec.template.spec.containers.0.image". Array elements use .N syntax.
	Path string `json:"path"`
	// Operator is the comparison operator.
	Operator WorkflowTriggerFieldOperator `json:"operator"`
	// Value to compare against. Required for equals, not_equals, changed_to, changed_from.
	Value string `json:"value,omitempty"`
}

// WorkflowTriggerFieldOperator defines comparison operators for field matching.
// +kubebuilder:validation:Enum=equals;not_equals;exists;not_exists;changed;changed_to;changed_from
type WorkflowTriggerFieldOperator string

const (
	FieldOperatorEquals      WorkflowTriggerFieldOperator = "equals"
	FieldOperatorNotEquals   WorkflowTriggerFieldOperator = "not_equals"
	FieldOperatorExists      WorkflowTriggerFieldOperator = "exists"
	FieldOperatorNotExists   WorkflowTriggerFieldOperator = "not_exists"
	FieldOperatorChanged     WorkflowTriggerFieldOperator = "changed"
	FieldOperatorChangedTo   WorkflowTriggerFieldOperator = "changed_to"
	FieldOperatorChangedFrom WorkflowTriggerFieldOperator = "changed_from"
)

// WorkflowTriggerWait defines stability gates before execution.
type WorkflowTriggerWait struct {
	// Conditions defines K8s resource conditions to wait for.
	Conditions *WorkflowTriggerWaitConditions `json:"conditions,omitempty"`
	// Probes defines HTTP health checks to wait for.
	Probes *WorkflowTriggerWaitProbes `json:"probes,omitempty"`
}

// WorkflowTriggerWaitConditions defines conditions to wait for before executing.
type WorkflowTriggerWaitConditions struct {
	// Items is the list of conditions to match.
	Items []WorkflowTriggerCondition `json:"items"`
	// Timeout is the maximum time in seconds to wait for conditions.
	Timeout int32 `json:"timeout,omitempty"`
	// Delay is the time in seconds between condition checks.
	Delay int32 `json:"delay,omitempty"`
}

// WorkflowTriggerCondition defines a single condition to match.
type WorkflowTriggerCondition struct {
	// Type is the condition type (e.g. Available, Ready).
	Type string `json:"type"`
	// Status is the expected condition status.
	Status *WorkflowTriggerConditionStatus `json:"status"`
	// Reason is the expected condition reason. Optional.
	Reason string `json:"reason,omitempty"`
	// TTL is the maximum age in seconds for the condition to be considered valid.
	TTL int32 `json:"ttl,omitempty"`
}

// WorkflowTriggerConditionStatus defines condition status values.
// +kubebuilder:validation:Enum=True;False;Unknown
type WorkflowTriggerConditionStatus string

const (
	WorkflowTriggerConditionStatusTrue    WorkflowTriggerConditionStatus = "True"
	WorkflowTriggerConditionStatusFalse   WorkflowTriggerConditionStatus = "False"
	WorkflowTriggerConditionStatusUnknown WorkflowTriggerConditionStatus = "Unknown"
)

// WorkflowTriggerWaitProbes defines HTTP probes to wait for before executing.
type WorkflowTriggerWaitProbes struct {
	// Items is the list of probes to check.
	Items []WorkflowTriggerProbe `json:"items"`
	// Timeout is the maximum time in seconds to wait for probes.
	Timeout int32 `json:"timeout,omitempty"`
	// Delay is the time in seconds between probe checks.
	Delay int32 `json:"delay,omitempty"`
}

// WorkflowTriggerProbe defines an HTTP probe.
type WorkflowTriggerProbe struct {
	// Scheme is the probe scheme (http or https). Default: http.
	Scheme string `json:"scheme,omitempty"`
	// Host is the probe host. Default: pod IP or service name.
	Host string `json:"host,omitempty"`
	// Path is the probe path. Default: /.
	Path string `json:"path,omitempty"`
	// Port is the probe port.
	Port int32 `json:"port,omitempty"`
	// Headers are HTTP headers to send with the probe.
	Headers map[string]string `json:"headers,omitempty"`
}

// WorkflowTriggerRun defines what workflow to execute and how.
type WorkflowTriggerRun struct {
	// Workflow identifies which workflow(s) to execute.
	Workflow WorkflowTriggerWorkflowSelector `json:"workflow"`
	// Target defines runner targeting (match/not/replicate).
	Target *commonv1.Target `json:"target,omitempty" expr:"include"`
	// Parameters defines config values and tags passed to the workflow.
	// Config values support the Testkube expression language: {{resource.spec.replicas}}
	Parameters *WorkflowTriggerRunParameters `json:"parameters,omitempty"`
	// ConcurrencyPolicy defines how concurrent executions are handled.
	// +kubebuilder:validation:Enum=allow;forbid;replace
	ConcurrencyPolicy string `json:"concurrencyPolicy,omitempty"`
	// Delay is the duration to wait before executing.
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Format:=duration
	Delay *metav1.Duration `json:"delay,omitempty"`
}

// WorkflowTriggerWorkflowSelector identifies which workflow(s) to execute.
type WorkflowTriggerWorkflowSelector struct {
	// Name matches a workflow by exact name.
	Name string `json:"name,omitempty"`
	// NameRegex matches workflows by name regex.
	NameRegex string `json:"nameRegex,omitempty"`
	// LabelSelector matches workflows by K8s label selector.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// WorkflowTriggerRunParameters defines config values and tags for execution.
type WorkflowTriggerRunParameters struct {
	// Config maps config variable names to values or expressions.
	Config map[string]string `json:"config,omitempty"`
	// Tags maps tag names to values or expressions.
	Tags map[string]string `json:"tags,omitempty"`
}

// WorkflowTriggerStatus defines the observed state of WorkflowTrigger.
type WorkflowTriggerStatus struct{}

//+kubebuilder:object:root=true

// WorkflowTriggerList contains a list of WorkflowTrigger
type WorkflowTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowTrigger `json:"items"`
}
