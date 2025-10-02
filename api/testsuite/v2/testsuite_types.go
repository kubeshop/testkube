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

package v2

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
	Schedule         string                     `json:"schedule,omitempty"`
	ExecutionRequest *TestSuiteExecutionRequest `json:"executionRequest,omitempty"`
}

type Variable commonv1.Variable

// TestSuiteStepSpec for particular type will have config for possible step types
type TestSuiteStepSpec struct {
	Type    TestSuiteStepType     `json:"type,omitempty"`
	Execute *TestSuiteStepExecute `json:"execute,omitempty"`
	Delay   *TestSuiteStepDelay   `json:"delay,omitempty"`
}

// TestSuiteStepType defines different type of test suite steps
// +kubebuilder:validation:Enum=execute;delay
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

// RunningContext for test or test suite execution
type RunningContext struct {
	// One of possible context types
	Type_ RunningContextType `json:"type"`
	// Context value depending from its type
	Context string `json:"context,omitempty"`
}

type RunningContextType string

const (
	RunningContextTypeUserCLI     RunningContextType = "user-cli"
	RunningContextTypeUserUI      RunningContextType = "user-ui"
	RunningContextTypeTestSuite   RunningContextType = "testsuite"
	RunningContextTypeTestTrigger RunningContextType = "testtrigger"
	RunningContextTypeScheduler   RunningContextType = "scheduler"
	RunningContextTypeEmpty       RunningContextType = ""
)

// TestSuiteExecutionRequest defines the execution request body
type TestSuiteExecutionRequest struct {
	// test execution custom name
	Name string `json:"name,omitempty"`
	// test kubernetes namespace (\"testkube\" when not set)
	Namespace string              `json:"namespace,omitempty"`
	Variables map[string]Variable `json:"variables,omitempty"`
	// secret uuid
	SecretUUID string `json:"secretUUID,omitempty"`
	// test suite labels
	Labels map[string]string `json:"labels,omitempty"`
	// execution labels
	ExecutionLabels map[string]string `json:"executionLabels,omitempty"`
	// whether to start execution sync or async
	Sync bool `json:"sync,omitempty"`
	// http proxy for executor containers
	HttpProxy string `json:"httpProxy,omitempty"`
	// https proxy for executor containers
	HttpsProxy string `json:"httpsProxy,omitempty"`
	// timeout for test suite execution
	Timeout        int32           `json:"timeout,omitempty"`
	RunningContext *RunningContext `json:"-"`
	// cron job template extensions
	CronJobTemplate string `json:"cronJobTemplate,omitempty"`
}

// +kubebuilder:validation:Enum=queued;running;passed;failed;aborting;aborted;timeout
type TestSuiteExecutionStatus string

// List of TestSuiteExecutionStatus
const (
	QUEUED_TestSuiteExecutionStatus   TestSuiteExecutionStatus = "queued"
	RUNNING_TestSuiteExecutionStatus  TestSuiteExecutionStatus = "running"
	PASSED_TestSuiteExecutionStatus   TestSuiteExecutionStatus = "passed"
	FAILED_TestSuiteExecutionStatus   TestSuiteExecutionStatus = "failed"
	ABORTING_TestSuiteExecutionStatus TestSuiteExecutionStatus = "aborting"
	ABORTED_TestSuiteExecutionStatus  TestSuiteExecutionStatus = "aborted"
	TIMEOUT_TestSuiteExecutionStatus  TestSuiteExecutionStatus = "timeout"
)

// test suite execution core
type TestSuiteExecutionCore struct {
	// execution id
	Id string `json:"id,omitempty"`
	// test suite execution start time
	StartTime metav1.Time `json:"startTime,omitempty"`
	// test suite execution end time
	EndTime metav1.Time               `json:"endTime,omitempty"`
	Status  *TestSuiteExecutionStatus `json:"status,omitempty"`
}

// TestSuiteStatus defines the observed state of TestSuite
type TestSuiteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// latest execution result
	LatestExecution *TestSuiteExecutionCore `json:"latestExecution,omitempty"`
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
