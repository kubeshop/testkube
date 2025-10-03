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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Variable commonv1.Variable

// test content request body
type TestContentRequest struct {
	Repository *RepositoryParameters `json:"repository,omitempty"`
}

// repository parameters for tests in git repositories
type RepositoryParameters struct {
	// branch/tag name for checkout
	Branch string `json:"branch,omitempty"`
	// commit id (sha) for checkout
	Commit string `json:"commit,omitempty"`
	// if needed we can checkout particular path (dir or file) in case of BIG/mono repositories
	Path string `json:"path,omitempty"`
	// if provided we checkout the whole repository and run test from this directory
	WorkingDir string `json:"workingDir,omitempty"`
}

// running context for test or test suite execution
type RunningContext struct {
	// One of possible context types
	Type_ RunningContextType `json:"type"`
	// Context value depending from its type
	Context string `json:"context,omitempty"`
}

// RunningContextType defines running context type
// +kubebuilder:validation:Enum=user-cli;user-ui;testsuite;testtrigger;scheduler;testexecution;testsuiteexecution
type RunningContextType string

const (
	RunningContextTypeUserCLI            RunningContextType = "user-cli"
	RunningContextTypeUserUI             RunningContextType = "user-ui"
	RunningContextTypeTestSuite          RunningContextType = "testsuite"
	RunningContextTypeTestTrigger        RunningContextType = "testtrigger"
	RunningContextTypeScheduler          RunningContextType = "scheduler"
	RunningContextTypeTestExecution      RunningContextType = "testexecution"
	RunningContextTypeTestSuiteExecution RunningContextType = "testsuiteexecution"
	RunningContextTypeEmpty              RunningContextType = ""
)

// test suite execution request body
type TestSuiteExecutionRequest struct {
	// test execution custom name
	Name string `json:"name,omitempty"`
	// test suite execution number
	Number int32 `json:"number,omitempty"`
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
	// duration in seconds the test suite may be active, until its stopped
	Timeout        int32               `json:"timeout,omitempty"`
	ContentRequest *TestContentRequest `json:"contentRequest,omitempty"`
	RunningContext *RunningContext     `json:"runningContext,omitempty"`
	// cron job template extensions
	CronJobTemplate string `json:"cronJobTemplate,omitempty"`
	// number of tests run in parallel
	ConcurrencyLevel int32 `json:"concurrencyLevel,omitempty"`
	// test suite execution name started the test suite execution
	TestSuiteExecutionName string `json:"testSuiteExecutionName,omitempty"`
	// whether webhooks should be disabled for this execution
	DisableWebhooks bool `json:"disableWebhooks,omitempty"`
}

type ObjectRef struct {
	// object kubernetes namespace
	Namespace string `json:"namespace,omitempty"`
	// object name
	Name string `json:"name"`
}

// TestSuiteExecutionSpec defines the desired state of TestSuiteExecution
type TestSuiteExecutionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	TestSuite        *ObjectRef                 `json:"testSuite"`
	ExecutionRequest *TestSuiteExecutionRequest `json:"executionRequest,omitempty"`
}

// TestSuiteExecutionStatus defines the observed state of TestSuiteExecution
type TestSuiteExecutionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LatestExecution *SuiteExecution `json:"latestExecution,omitempty"`
	// test status execution generation
	Generation int64 `json:"generation,omitempty"`
}

// SuiteExecutions data
type SuiteExecution struct {
	// execution id
	Id string `json:"id"`
	// execution name
	Name      string                `json:"name"`
	TestSuite *ObjectRef            `json:"testSuite"`
	Status    *SuiteExecutionStatus `json:"status,omitempty"`
	// Environment variables passed to executor.
	// Deprecated: use Basic Variables instead
	Envs      map[string]string   `json:"envs,omitempty"`
	Variables map[string]Variable `json:"variables,omitempty"`
	// secret uuid
	SecretUUID string `json:"secretUUID,omitempty"`
	// test start time
	StartTime metav1.Time `json:"startTime,omitempty"`
	// test end time
	EndTime metav1.Time `json:"endTime,omitempty"`
	// test duration
	Duration string `json:"duration,omitempty"`
	// test duration in ms
	DurationMs int32 `json:"durationMs,omitempty"`
	// steps execution results
	StepResults []TestSuiteStepExecutionResultV2 `json:"stepResults,omitempty"`
	// batch steps execution results
	ExecuteStepResults []TestSuiteBatchStepExecutionResult `json:"executeStepResults,omitempty"`
	// test suite labels
	Labels         map[string]string `json:"labels,omitempty"`
	RunningContext *RunningContext   `json:"runningContext,omitempty"`
	// test suite execution name started the test suite execution
	TestSuiteExecutionName string `json:"testSuiteExecutionName,omitempty"`
	// whether webhooks should be disabled for this execution
	DisableWebhooks bool `json:"disableWebhooks,omitempty"`
}

// execution result returned from executor
type TestSuiteStepExecutionResultV2 struct {
	Step      *TestSuiteStepV2 `json:"step,omitempty"`
	Test      *ObjectRef       `json:"test,omitempty"`
	Execution *Execution       `json:"execution,omitempty"`
}

type TestSuiteStepV2 struct {
	StopTestOnFailure bool                        `json:"stopTestOnFailure"`
	Execute           *TestSuiteStepExecuteTestV2 `json:"execute,omitempty"`
	Delay             *TestSuiteStepDelayV2       `json:"delay,omitempty"`
}

type TestSuiteStepExecuteTestV2 struct {
	// object kubernetes namespace
	Namespace string `json:"namespace,omitempty"`
	// object name
	Name string `json:"name"`
}

type TestSuiteStepDelayV2 struct {
	// delay duration in milliseconds
	Duration int32 `json:"duration"`
}

// ArgsModeType defines args mode type
// +kubebuilder:validation:Enum=append;override;replace
type ArgsModeType string

const (
	// ArgsModeTypeAppend for append args mode
	ArgsModeTypeAppend ArgsModeType = "append"
	// ArgsModeTypeOverride for override args mode
	ArgsModeTypeOverride ArgsModeType = "override"
	// ArgsModeTypeReplace for replace args mode
	ArgsModeTypeReplace ArgsModeType = "replace"
)

// pod request body
type PodRequest struct {
	Resources *PodResourcesRequest `json:"resources,omitempty"`
	// pod template extensions
	PodTemplate string `json:"podTemplate,omitempty"`
	// name of the template resource
	PodTemplateReference string `json:"podTemplateReference,omitempty"`
}

// pod resources request specification
type PodResourcesRequest struct {
	Requests *ResourceRequest `json:"requests,omitempty"`
	Limits   *ResourceRequest `json:"limits,omitempty"`
}

// resource request specification
type ResourceRequest struct {
	// requested cpu units
	Cpu string `json:"cpu,omitempty"`
	// requested memory units
	Memory string `json:"memory,omitempty"`
}

// test execution
type Execution struct {
	// execution id
	Id string `json:"id,omitempty"`
	// unique test name (CRD Test name)
	TestName string `json:"testName,omitempty"`
	// unique test suite name (CRD Test suite name), if it's run as a part of test suite
	TestSuiteName string `json:"testSuiteName,omitempty"`
	// test namespace
	TestNamespace string `json:"testNamespace,omitempty"`
	// test type e.g. postman/collection
	TestType string `json:"testType,omitempty"`
	// execution name
	Name string `json:"name,omitempty"`
	// execution number
	Number int32 `json:"number,omitempty"`
	// Environment variables passed to executor.
	// Deprecated: use Basic Variables instead
	Envs map[string]string `json:"envs,omitempty"`
	// executor image command
	Command []string `json:"command,omitempty"`
	// additional arguments/flags passed to executor binary
	Args []string `json:"args,omitempty"`
	// usage mode for arguments
	ArgsMode  ArgsModeType        `json:"args_mode,omitempty"`
	Variables map[string]Variable `json:"variables,omitempty"`
	// in case the variables file is too big, it will be uploaded to storage
	IsVariablesFileUploaded bool `json:"isVariablesFileUploaded,omitempty"`
	// variables file content - need to be in format for particular executor (e.g. postman envs file)
	VariablesFile string `json:"variablesFile,omitempty"`
	// test secret uuid
	TestSecretUUID string `json:"testSecretUUID,omitempty"`
	// test suite secret uuid, if it's run as a part of test suite
	TestSuiteSecretUUID string       `json:"testSuiteSecretUUID,omitempty"`
	Content             *TestContent `json:"content,omitempty"`
	// test start time
	StartTime metav1.Time `json:"startTime,omitempty"`
	// test end time
	EndTime metav1.Time `json:"endTime,omitempty"`
	// test duration
	Duration string `json:"duration,omitempty"`
	// test duration in milliseconds
	DurationMs      int32            `json:"durationMs,omitempty"`
	ExecutionResult *ExecutionResult `json:"executionResult,omitempty"`
	// test and execution labels
	Labels map[string]string `json:"labels,omitempty"`
	// list of file paths that need to be copied into the test from uploads
	Uploads []string `json:"uploads,omitempty"`
	// minio bucket name to get uploads from
	BucketName      string           `json:"bucketName,omitempty"`
	ArtifactRequest *ArtifactRequest `json:"artifactRequest,omitempty"`
	// script to run before test execution
	PreRunScript string `json:"preRunScript,omitempty"`
	// script to run after test execution
	PostRunScript string `json:"postRunScript,omitempty"`
	// execute post run script before scraping (prebuilt executor only)
	ExecutePostRunScriptBeforeScraping bool `json:"executePostRunScriptBeforeScraping,omitempty"`
	// run scripts using source command (container executor only)
	SourceScripts  bool            `json:"sourceScripts,omitempty"`
	RunningContext *RunningContext `json:"runningContext,omitempty"`
	// shell used in container executor
	ContainerShell string `json:"containerShell,omitempty"`
	// test execution name started the test execution
	TestExecutionName string      `json:"testExecutionName,omitempty"`
	SlavePodRequest   *PodRequest `json:"slavePodRequest,omitempty"`
	// namespace for test execution (Pro edition only)
	ExecutionNamespace string `json:"executionNamespace,omitempty"`
	// whether webhooks should be disabled for this execution
	DisableWebhooks bool `json:"disableWebhooks,omitempty"`
}

// artifact request body with test artifacts
type ArtifactRequest struct {
	// artifact storage class name for container executor
	StorageClassName string `json:"storageClassName,omitempty"`
	// artifact volume mount path for container executor
	VolumeMountPath string `json:"volumeMountPath,omitempty"`
	// artifact directories for scraping
	Dirs []string `json:"dirs,omitempty"`
	// regexp to filter scraped artifacts, single or comma separated
	Masks []string `json:"masks,omitempty"`
	// artifact bucket storage
	StorageBucket string `json:"storageBucket,omitempty"`
	// don't use a separate folder for execution artifacts
	OmitFolderPerExecution bool `json:"omitFolderPerExecution,omitempty"`
	// whether to share volume between pods
	SharedBetweenPods bool `json:"sharedBetweenPods,omitempty"`
	// whether to use default storage class name
	UseDefaultStorageClassName bool `json:"useDefaultStorageClassName,omitempty"`
	// run scraper as pod sidecar container
	SidecarScraper bool `json:"sidecarScraper,omitempty"`
}

// TestContent defines test content
type TestContent struct {
	// test type
	Type_ TestContentType `json:"type,omitempty"`
	// repository of test content
	Repository *Repository `json:"repository,omitempty"`
	// test content body
	Data string `json:"data,omitempty"`
	// uri of test content
	Uri string `json:"uri,omitempty"`
}

// +kubebuilder:validation:Enum=string;file-uri;git-file;git-dir;git
type TestContentType string

const (
	TestContentTypeString  TestContentType = "string"
	TestContentTypeFileURI TestContentType = "file-uri"
	// Deprecated: use git instead
	TestContentTypeGitFile TestContentType = "git-file"
	// Deprecated: use git instead
	TestContentTypeGitDir TestContentType = "git-dir"
	TestContentTypeGit    TestContentType = "git"
)

// Testkube internal reference for secret storage in Kubernetes secrets
type SecretRef struct {
	// object kubernetes namespace
	Namespace string `json:"namespace,omitempty"`
	// object name
	Name string `json:"name"`
	// object key
	Key string `json:"key"`
}

// Repository represents VCS repo, currently we're handling Git only
type Repository struct {
	// VCS repository type
	Type_ string `json:"type,omitempty"`
	// uri of content file or git directory
	Uri string `json:"uri,omitempty"`
	// branch/tag name for checkout
	Branch string `json:"branch,omitempty"`
	// commit id (sha) for checkout
	Commit string `json:"commit,omitempty"`
	// if needed we can checkout particular path (dir or file) in case of BIG/mono repositories
	Path           string     `json:"path,omitempty"`
	UsernameSecret *SecretRef `json:"usernameSecret,omitempty"`
	TokenSecret    *SecretRef `json:"tokenSecret,omitempty"`
	// git auth certificate secret for private repositories
	CertificateSecret string `json:"certificateSecret,omitempty"`
	// if provided we checkout the whole repository and run test from this directory
	WorkingDir string `json:"workingDir,omitempty"`
	// auth type for git requests
	AuthType GitAuthType `json:"authType,omitempty"`
}

// GitAuthType defines git auth type
// +kubebuilder:validation:Enum=basic;header
type GitAuthType string

const (
	// GitAuthTypeBasic for git basic auth requests
	GitAuthTypeBasic GitAuthType = "basic"
	// GitAuthTypeHeader for git header auth requests
	GitAuthTypeHeader GitAuthType = "header"
)

// execution result returned from executor
type ExecutionResult struct {
	Status *ExecutionStatus `json:"status"`
	// error message when status is error, separate to output as output can be partial in case of error
	ErrorMessage string `json:"errorMessage,omitempty"`
	// execution steps (for collection of requests)
	Steps   []ExecutionStepResult   `json:"steps,omitempty"`
	Reports *ExecutionResultReports `json:"reports,omitempty"`
}

// execution result data
type ExecutionStepResult struct {
	// step name
	Name     string `json:"name"`
	Duration string `json:"duration,omitempty"`
	// execution step status
	Status           string            `json:"status"`
	AssertionResults []AssertionResult `json:"assertionResults,omitempty"`
}

// execution result data
type AssertionResult struct {
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type ExecutionResultReports struct {
	Junit string `json:"junit,omitempty"`
}

// +kubebuilder:validation:Enum=queued;running;passed;failed;aborted;timeout
type ExecutionStatus string

// List of ExecutionStatus
const (
	QUEUED_ExecutionStatus  ExecutionStatus = "queued"
	RUNNING_ExecutionStatus ExecutionStatus = "running"
	PASSED_ExecutionStatus  ExecutionStatus = "passed"
	FAILED_ExecutionStatus  ExecutionStatus = "failed"
	ABORTED_ExecutionStatus ExecutionStatus = "aborted"
	TIMEOUT_ExecutionStatus ExecutionStatus = "timeout"
)

// execution result returned from executor
type TestSuiteBatchStepExecutionResult struct {
	Step    *TestSuiteBatchStep            `json:"step,omitempty"`
	Execute []TestSuiteStepExecutionResult `json:"execute,omitempty"`
	// step start time
	StartTime metav1.Time `json:"startTime,omitempty"`
	// step end time
	EndTime metav1.Time `json:"endTime,omitempty"`
	// step duration
	Duration string `json:"duration,omitempty"`
}

// set of steps run in parallel
type TestSuiteBatchStep struct {
	StopOnFailure bool            `json:"stopOnFailure"`
	Execute       []TestSuiteStep `json:"execute,omitempty"`
}

type TestSuiteStep struct {
	// object name
	Test string `json:"test,omitempty"`
	// delay duration in time units
	Delay string `json:"delay,omitempty"`
}

// execution result returned from executor
type TestSuiteStepExecutionResult struct {
	Step      *TestSuiteStep `json:"step,omitempty"`
	Test      *ObjectRef     `json:"test,omitempty"`
	Execution *Execution     `json:"execution,omitempty"`
}

// +kubebuilder:validation:Enum=queued;running;passed;failed;aborting;aborted;timeout
type SuiteExecutionStatus string

// List of SuiteExecutionStatus
const (
	QUEUED_SuiteExecutionStatus   SuiteExecutionStatus = "queued"
	RUNNING_SuiteExecutionStatus  SuiteExecutionStatus = "running"
	PASSED_SuiteExecutionStatus   SuiteExecutionStatus = "passed"
	FAILED_SuiteExecutionStatus   SuiteExecutionStatus = "failed"
	ABORTING_SuiteExecutionStatus SuiteExecutionStatus = "aborting"
	ABORTED_SuiteExecutionStatus  SuiteExecutionStatus = "aborted"
	TIMEOUT_SuiteExecutionStatus  SuiteExecutionStatus = "timeout"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TestSuiteExecution is the Schema for the testsuiteexecutions API
type TestSuiteExecution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestSuiteExecutionSpec   `json:"spec,omitempty"`
	Status TestSuiteExecutionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TestSuiteExecutionList contains a list of TestSuiteExecution
type TestSuiteExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestSuiteExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestSuiteExecution{}, &TestSuiteExecutionList{})
}
