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

package v3

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

	// Before batch steps is list of batch tests which will be sequentially orchestrated for parallel tests in each batch
	Before []TestSuiteBatchStep `json:"before,omitempty"`
	// Batch steps is list of batch tests which will be sequentially orchestrated for parallel tests in each batch
	Steps []TestSuiteBatchStep `json:"steps,omitempty"`
	// After batch steps is list of batch tests which will be sequentially orchestrated for parallel tests in each batch
	After []TestSuiteBatchStep `json:"after,omitempty"`

	Repeats     int    `json:"repeats,omitempty"`
	Description string `json:"description,omitempty"`
	// schedule in cron job format for scheduled test execution
	Schedule         string                     `json:"schedule,omitempty"`
	ExecutionRequest *TestSuiteExecutionRequest `json:"executionRequest,omitempty"`
}

type Variable commonv1.Variable

// TestSuiteStepSpec for particular type will have config for possible step types
type TestSuiteStepSpec struct {
	// object name
	Test string `json:"test,omitempty"`
	// delay duration in time units
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Format:=duration
	Delay            metav1.Duration                `json:"delay,omitempty"`
	ExecutionRequest *TestSuiteStepExecutionRequest `json:"executionRequest,omitempty"`
}

// options to download artifacts from previous steps
type DownloadArtifactOptions struct {
	AllPreviousSteps bool `json:"allPreviousSteps,omitempty"`
	// previous step numbers starting from 1
	PreviousStepNumbers []int32 `json:"previousStepNumbers,omitempty"`
	// previous test names
	PreviousTestNames []string `json:"previousTestNames,omitempty"`
}

// TestSuiteBatchStep is set of steps run in parallel
type TestSuiteBatchStep struct {
	StopOnFailure     bool                     `json:"stopOnFailure"`
	DownloadArtifacts *DownloadArtifactOptions `json:"downloadArtifacts,omitempty"`
	Execute           []TestSuiteStepSpec      `json:"execute,omitempty"`
}

// RunningContext for test or test suite execution
type RunningContext struct {
	// One of possible context types
	Type_ RunningContextType `json:"type"`
	// Context value which depends from its type
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

// test suite execution request body
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
	// job template extensions
	JobTemplate string `json:"jobTemplate,omitempty"`
	// name of the template resource
	JobTemplateReference string `json:"jobTemplateReference,omitempty"`
	// cron job template extensions
	CronJobTemplate string `json:"cronJobTemplate,omitempty"`
	// name of the template resource
	CronJobTemplateReference string `json:"cronJobTemplateReference,omitempty"`
	// scraper template extensions
	ScraperTemplate string `json:"scraperTemplate,omitempty"`
	// name of the template resource
	ScraperTemplateReference string `json:"scraperTemplateReference,omitempty"`
	// pvc template extensions
	PvcTemplate string `json:"pvcTemplate,omitempty"`
	// name of the template resource
	PvcTemplateReference string `json:"pvcTemplateReference,omitempty"`
	// whether webhooks should be called on execution
	// Deprecated: field is not used
	DisableWebhooks bool `json:"disableWebhooks,omitempty"`
}

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

// TestSuiteExecutionCore defines the observed state of TestSuiteExecution
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
//+kubebuilder:storageversion

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

type ArgsModeType commonv1.ArgsModeType

// TestSuiteStepExecutionRequest contains parameters to be used by the executions.
// These fields will be passed to the execution when a Test Suite is queued for execution.
// TestSuiteStepExecutionRequest parameters have the highest priority. They override the
// values coming from Test Suites, Tests, and Test Executions.
// +kubebuilder:object:generate=true
type TestSuiteStepExecutionRequest struct {
	// test execution labels
	ExecutionLabels map[string]string   `json:"executionLabels,omitempty"`
	Variables       map[string]Variable `json:"variables,omitempty"`
	// additional executor binary arguments
	Args []string `json:"args,omitempty"`
	// usage mode for arguments
	ArgsMode ArgsModeType `json:"argsMode,omitempty"`
	// executor binary command
	Command []string `json:"command,omitempty"`
	// whether to start execution sync or async
	Sync bool `json:"sync,omitempty"`
	// http proxy for executor containers
	HttpProxy string `json:"httpProxy,omitempty"`
	// https proxy for executor containers
	HttpsProxy string `json:"httpsProxy,omitempty"`
	// negative test will fail the execution if it is a success and it will succeed if it is a failure
	NegativeTest bool `json:"negativeTest,omitempty"`
	// job template extensions
	JobTemplate string `json:"jobTemplate,omitempty"`
	// job template extensions reference
	JobTemplateReference string `json:"jobTemplateReference,omitempty"`
	// cron job template extensions
	CronJobTemplate string `json:"cronJobTemplate,omitempty"`
	// cron job template extensions reference
	CronJobTemplateReference string `json:"cronJobTemplateReference,omitempty"`
	// scraper template extensions
	ScraperTemplate string `json:"scraperTemplate,omitempty"`
	// scraper template extensions reference
	ScraperTemplateReference string `json:"scraperTemplateReference,omitempty"`
	// pvc template extensions
	PvcTemplate string `json:"pvcTemplate,omitempty"`
	// pvc template extensions reference
	PvcTemplateReference string                   `json:"pvcTemplateReference,omitempty"`
	RunningContext       *commonv1.RunningContext `json:"runningContext,omitempty"`
	// whether webhooks should be called on execution
	// Deprecated: field is not used
	DisableWebhooks bool `json:"disableWebhooks,omitempty"`
}
