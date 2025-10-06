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

// WebhookSpec defines the desired state of Webhook
type WebhookSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Uri is address where webhook should be made (golang template supported)
	Uri string `json:"uri,omitempty"`
	// Events declare list if events on which webhook should be called
	Events []EventType `json:"events,omitempty"`
	// Labels to filter for tests and test suites
	Selector string `json:"selector,omitempty"`
	// will load the generated payload for notification inside the object
	PayloadObjectField string `json:"payloadObjectField,omitempty"`
	// golang based template for notification payload
	PayloadTemplate string `json:"payloadTemplate,omitempty"`
	// name of the template resource
	PayloadTemplateReference string `json:"payloadTemplateReference,omitempty"`
	// webhook headers (golang template supported)
	Headers map[string]string `json:"headers,omitempty"`
	// Disabled will disable the webhook
	Disabled bool `json:"disabled,omitempty"`
	// OnStateChange will trigger the webhook only when the result of the current execution differs from the previous result of the same test/test suite/workflow
	// Deprecated: field is not used
	OnStateChange bool `json:"onStateChange,omitempty"`
	// webhook configuration
	Config map[string]WebhookConfigValue `json:"config,omitempty"`
	// webhook parameters
	Parameters []WebhookParameterSchema `json:"parameters,omitempty"`
	// webhook template reference
	WebhookTemplateRef *WebhookTemplateRef `json:"webhookTemplateRef,omitempty"`
}

// WebhookTemplateSpec defines the desired state of Webhook Template
type WebhookTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Uri is address where webhook should be made (golang template supported)
	Uri string `json:"uri,omitempty"`
	// Events declare list if events on which webhook should be called
	Events []EventType `json:"events,omitempty"`
	// Labels to filter for tests and test suites
	Selector string `json:"selector,omitempty"`
	// will load the generated payload for notification inside the object
	PayloadObjectField string `json:"payloadObjectField,omitempty"`
	// golang based template for notification payload
	PayloadTemplate string `json:"payloadTemplate,omitempty"`
	// name of the template resource
	PayloadTemplateReference string `json:"payloadTemplateReference,omitempty"`
	// webhook headers (golang template supported)
	Headers map[string]string `json:"headers,omitempty"`
	// Disabled will disable the webhook
	Disabled bool `json:"disabled,omitempty"`
	// webhook configuration
	Config map[string]WebhookConfigValue `json:"config,omitempty"`
	// webhook parameters
	Parameters []WebhookParameterSchema `json:"parameters,omitempty"`
}

// webhook parameter schema
type WebhookParameterSchema struct {
	// unique parameter name
	Name string `json:"name"`
	// description for the parameter
	Description string `json:"description,omitempty"`
	// whether parameter is required
	Required bool `json:"required,omitempty"`
	// example value for the parameter
	Example string `json:"example,omitempty"`
	// default parameter value
	Default_ *string `json:"default,omitempty"`
	// regular expression to match
	Pattern string `json:"pattern,omitempty"`
}

// webhook template reference
type WebhookTemplateRef struct {
	// webhook template name to include
	Name string `json:"name"`
}

// webhook configuration value
type WebhookConfigValue struct {
	// public value to use in webhook template
	Value *string `json:"value,omitempty"`
	// private value stored in secret to use in webhook template
	Secret *SecretRef `json:"secret,omitempty"`
}

// Testkube internal reference for secret storage in Kubernetes secrets
type SecretRef struct {
	// object kubernetes namespace
	Namespace string `json:"namespace,omitempty"`
	// object name
	Name string `json:"name"`
	// object key
	Key string `json:"key"`
}

// +kubebuilder:validation:Enum=start-test;end-test-success;end-test-failed;end-test-aborted;end-test-timeout;become-test-up;become-test-down;become-test-failed;become-test-aborted;become-test-timeout;start-testsuite;end-testsuite-success;end-testsuite-failed;end-testsuite-aborted;end-testsuite-timeout;become-testsuite-up;become-testsuite-down;become-testsuite-failed;become-testsuite-aborted;become-testsuite-timeout;start-testworkflow;queue-testworkflow;end-testworkflow-success;end-testworkflow-failed;end-testworkflow-aborted;end-testworkflow-canceled;end-testworkflow-not-passed;become-testworkflow-up;become-testworkflow-down;become-testworkflow-failed;become-testworkflow-aborted;become-testworkflow-canceled;become-testworkflow-not-passed
type EventType string

// List of EventType
const (
	START_TEST_EventType                     EventType = "start-test"
	END_TEST_SUCCESS_EventType               EventType = "end-test-success"
	END_TEST_FAILED_EventType                EventType = "end-test-failed"
	END_TEST_ABORTED_EventType               EventType = "end-test-aborted"
	END_TEST_TIMEOUT_EventType               EventType = "end-test-timeout"
	BECOME_TEST_UP_EventType                 EventType = "become-test-up"
	BECOME_TEST_DOWN_EventType               EventType = "become-test-down"
	BECOME_TEST_FAILED_EventType             EventType = "become-test-failed"
	BECOME_TEST_ABORTED_EventType            EventType = "become-test-aborted"
	BECOME_TEST_TIMEOUT_EventType            EventType = "become-test-timeout"
	START_TESTSUITE_EventType                EventType = "start-testsuite"
	END_TESTSUITE_SUCCESS_EventType          EventType = "end-testsuite-success"
	END_TESTSUITE_FAILED_EventType           EventType = "end-testsuite-failed"
	END_TESTSUITE_ABORTED_EventType          EventType = "end-testsuite-aborted"
	END_TESTSUITE_TIMEOUT_EventType          EventType = "end-testsuite-timeout"
	BECOME_TESTSUITE_UP_EventType            EventType = "become-testsuite-up"
	BECOME_TESTSUITE_DOWN_EventType          EventType = "become-testsuite-down"
	BECOME_TESTSUITE_FAILED_EventType        EventType = "become-testsuite-failed"
	BECOME_TESTSUITE_ABORTED_EventType       EventType = "become-testsuite-aborted"
	BECOME_TESTSUITE_TIMEOUT_EventType       EventType = "become-testsuite-timeout"
	START_TESTWORKFLOW_EventType             EventType = "start-testworkflow"
	QUEUE_TESTWORKFLOW_EventType             EventType = "queue-testworkflow"
	END_TESTWORKFLOW_SUCCESS_EventType       EventType = "end-testworkflow-success"
	END_TESTWORKFLOW_FAILED_EventType        EventType = "end-testworkflow-failed"
	END_TESTWORKFLOW_ABORTED_EventType       EventType = "end-testworkflow-aborted"
	END_TESTWORKFLOW_CANCELED_EventType      EventType = "end-testworkflow-canceled"
	END_TESTWORKFLOW_NOT_PASSED_EventType    EventType = "end-testworkflow-not-passed"
	BECOME_TESTWORKFLOW_UP_EventType         EventType = "become-testworkflow-up"
	BECOME_TESTWORKFLOW_DOWN_EventType       EventType = "become-testworkflow-down"
	BECOME_TESTWORKFLOW_FAILED_EventType     EventType = "become-testworkflow-failed"
	BECOME_TESTWORKFLOW_ABORTED_EventType    EventType = "become-testworkflow-aborted"
	BECOME_TESTWORKFLOW_CANCELED_EventType   EventType = "become-testworkflow-canceled"
	BECOME_TESTWORKFLOW_NOT_PASSED_EventType EventType = "become-testworkflow-not-passed"
)

// WebhookStatus defines the observed state of Webhook
type WebhookStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// WebhookTemplateStatus defines the observed state of Webhook Template
type WebhookTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Webhook is the Schema for the webhooks API
type Webhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookSpec   `json:"spec,omitempty"`
	Status WebhookStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WebhookTemplate is the Schema for the webhook templates API
type WebhookTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookTemplateSpec   `json:"spec,omitempty"`
	Status WebhookTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WebhookList contains a list of Webhook
type WebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Webhook `json:"items"`
}

//+kubebuilder:object:root=true

// WebhookTemplateList contains a list of Webhook Template
type WebhookTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebhookTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Webhook{}, &WebhookList{})
	SchemeBuilder.Register(&WebhookTemplate{}, &WebhookTemplateList{})
}
