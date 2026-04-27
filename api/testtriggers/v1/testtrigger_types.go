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

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TestTrigger is the Schema for the testtriggers API
// +kubebuilder:printcolumn:name="Resource",type=string,JSONPath=`.spec.resource`
// +kubebuilder:printcolumn:name="Event",type=string,JSONPath=`.spec.event`
// +kubebuilder:printcolumn:name="Execution",type=string,JSONPath=`.spec.execution`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type TestTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestTriggerSpec   `json:"spec,omitempty"`
	Status TestTriggerStatus `json:"status,omitempty"`
}

// TestTriggerSpec defines the desired state of TestTrigger
type TestTriggerSpec struct {
	// Selector is used to select events which trigger an action
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
	// For which Resource do we monitor Event which triggers an Action on certain conditions
	// +optional
	Resource TestTriggerResource `json:"resource,omitempty"`
	// ResourceRef specifies a resource to watch by Group/Version/Kind.
	// Works for any K8s resource including CRDs. Mutually exclusive with Resource.
	ResourceRef *TestTriggerResourceRef `json:"resourceRef,omitempty"`
	// ResourceSelector identifies which Kubernetes Objects should be watched
	// +optional
	ResourceSelector TestTriggerSelector `json:"resourceSelector,omitempty"`
	// On which Event for a Resource should an Action be triggered
	Event TestTriggerEvent `json:"event"`
	// Match filters which object changes fire the trigger.
	// Each entry evaluates a dot-path on the watched object (e.g. ".status.currentStepIndex",
	// ".spec.template.spec.containers.0.image") with an operator. All entries must pass (AND logic).
	Match []workflowtriggersv1.WorkflowTriggerFieldCondition `json:"match,omitempty"`
	// ListenerAgentIds pins this trigger to specific listener agent(s) — the
	// agents that watch the cluster for matching events and fire the trigger.
	// When empty, every listener-capable agent in the environment that picks
	// up the CRD will fire it (broadcast). Required when match[] is set,
	// because schema-aware match validation is only sound when the firing
	// listener is known at create time. The picker and validator ran against
	// that listener's cluster-resources snapshot, and a different listener
	// may have a different CRD inventory or RBAC.
	ListenerAgentIds []string `json:"listenerAgentIds,omitempty"`
	// What resource conditions should be matched
	ConditionSpec *TestTriggerConditionSpec `json:"conditionSpec,omitempty"`
	// What resource probes should be matched
	ProbeSpec *TestTriggerProbeSpec `json:"probeSpec,omitempty"`
	// ContentSelector identifies which content should be watched for changes
	ContentSelector *TestTriggerContentSelector `json:"contentSelector,omitempty"`
	// Action represents what needs to be executed for selected Execution
	Action           TestTriggerAction            `json:"action"`
	ActionParameters *TestTriggerActionParameters `json:"actionParameters,omitempty"`
	// Execution identifies for which test execution should an Action be executed
	Execution TestTriggerExecution `json:"execution"`
	// TestSelector identifies on which Testkube Kubernetes Objects an Action should be taken
	TestSelector TestTriggerSelector `json:"testSelector"`
	// ConcurrencyPolicy defines concurrency policy for selected Execution
	ConcurrencyPolicy TestTriggerConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`
	// Delay is a duration string which specifies how long should the test be delayed after a trigger is matched
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Format:=duration
	Delay *metav1.Duration `json:"delay,omitempty"`
	// whether test trigger is disabled
	Disabled bool `json:"disabled,omitempty"`
}

// TestTriggerResource defines resource for test triggers
// +kubebuilder:validation:Enum=pod;deployment;statefulset;daemonset;service;ingress;event;configmap;content
type TestTriggerResource string

// List of TestTriggerResources
const (
	TestTriggerResourcePod         TestTriggerResource = "pod"
	TestTriggerResourceDeployment  TestTriggerResource = "deployment"
	TestTriggerResourceStatefulSet TestTriggerResource = "statefulset"
	TestTriggerResourceDaemonSet   TestTriggerResource = "daemonset"
	TestTriggerResourceService     TestTriggerResource = "service"
	TestTriggerResourceIngress     TestTriggerResource = "ingress"
	TestTriggerResourceEvent       TestTriggerResource = "event"
	TestTriggerResourceConfigMap   TestTriggerResource = "configmap"
	TestTriggerResourceContent     TestTriggerResource = "content"
)

// TestTriggerResourceRef identifies a K8s resource by GVK.
type TestTriggerResourceRef struct {
	// Group is the API group (empty for core resources like Pod, Service).
	Group string `json:"group,omitempty"`
	// Version is the API version.
	Version string `json:"version,omitempty"`
	// Kind is the resource kind (e.g. Deployment, KafkaTopic).
	Kind string `json:"kind"`
}

// TestTriggerEvent defines event for test triggers
// +kubebuilder:validation:Enum=created;modified;deleted;git-push;git-tag-push;git-pull-request;deployment-scale-update;deployment-image-update;deployment-env-update;deployment-containers-modified;deployment-generation-modified;deployment-resource-modified;event-start-test;event-end-test-success;event-end-test-failed;event-end-test-aborted;event-end-test-timeout;event-start-testsuite;event-end-testsuite-success;event-end-testsuite-failed;event-end-testsuite-aborted;event-end-testsuite-timeout;event-queue-testworkflow;event-start-testworkflow;event-end-testworkflow-success;event-end-testworkflow-failed;event-end-testworkflow-aborted;event-end-testworkflow-canceled;event-end-testworkflow-not-passed;event-created;event-updated;event-deleted
type TestTriggerEvent string

// List of TestTriggerEvents
const (
	TestTriggerEventCreated                       TestTriggerEvent = "created"
	TestTriggerEventModified                      TestTriggerEvent = "modified"
	TestTriggerEventDeleted                       TestTriggerEvent = "deleted"
	TestTriggerEventGitPush                       TestTriggerEvent = "git-push"
	TestTriggerEventGitTagPush                    TestTriggerEvent = "git-tag-push"
	TestTriggerEventGitPullRequest                TestTriggerEvent = "git-pull-request"
	TestTriggerCauseDeploymentScaleUpdate         TestTriggerEvent = "deployment-scale-update"
	TestTriggerCauseDeploymentImageUpdate         TestTriggerEvent = "deployment-image-update"
	TestTriggerCauseDeploymentEnvUpdate           TestTriggerEvent = "deployment-env-update"
	TestTriggerCauseDeploymentContainersModified  TestTriggerEvent = "deployment-containers-modified"
	TestTriggerCauseDeploymentGenerationModified  TestTriggerEvent = "deployment-generation-modified"
	TestTriggerCauseDeploymentResourceModified    TestTriggerEvent = "deployment-resource-modified"
	TestTriggerCauseEventStartTest                TestTriggerEvent = "event-start-test"
	TestTriggerCauseEventEndTestSuccess           TestTriggerEvent = "event-end-test-success"
	TestTriggerCauseEventEndTestFailed            TestTriggerEvent = "event-end-test-failed"
	TestTriggerCauseEventEndTestAborted           TestTriggerEvent = "event-end-test-aborted"
	TestTriggerCauseEventEndTestTimeout           TestTriggerEvent = "event-end-test-timeout"
	TestTriggerCauseEventStartTestSuite           TestTriggerEvent = "event-start-testsuite"
	TestTriggerCauseEventEndTestSuiteSuccess      TestTriggerEvent = "event-end-testsuite-success"
	TestTriggerCauseEventEndTestSuiteFailed       TestTriggerEvent = "event-end-testsuite-failed"
	TestTriggerCauseEventEndTestSuiteAborted      TestTriggerEvent = "event-end-testsuite-aborted"
	TestTriggerCauseEventEndTestSuiteTimeout      TestTriggerEvent = "event-end-testsuite-timeout"
	TestTriggerCauseEventQueueTestWorkflow        TestTriggerEvent = "event-queue-testworkflow"
	TestTriggerCauseEventStartTestWorkflow        TestTriggerEvent = "event-start-testworkflow"
	TestTriggerCauseEventEndTestWorkflowSuccess   TestTriggerEvent = "event-end-testworkflow-success"
	TestTriggerCauseEventEndTestWorkflowFailed    TestTriggerEvent = "event-end-testworkflow-failed"
	TestTriggerCauseEventEndTestWorkflowAborted   TestTriggerEvent = "event-end-testworkflow-aborted"
	TestTriggerCauseEventEndTestWorkflowCanceled  TestTriggerEvent = "event-end-testworkflow-canceled"
	TestTriggerCauseEventEndTestWorkflowNotPassed TestTriggerEvent = "event-end-testworkflow-not-passed"
	TestTriggerCauseEventCreated                  TestTriggerEvent = "event-created"
	TestTriggerCauseEventUpdated                  TestTriggerEvent = "event-updated"
	TestTriggerCauseEventDeleted                  TestTriggerEvent = "event-deleted"
)

// TestTriggerAction defines action for test triggers
// +kubebuilder:validation:Enum=run
type TestTriggerAction string

// List of TestTriggerAction
const (
	TestTriggerActionRun TestTriggerAction = "run"
)

// TestTriggerExecution defines execution for test triggers
// +kubebuilder:validation:Enum=test;testsuite;testworkflow
type TestTriggerExecution string

// List of TestTriggerExecution
const (
	TestTriggerExecutionTest         TestTriggerExecution = "test"
	TestTriggerExecutionTestsuite    TestTriggerExecution = "testsuite"
	TestTriggerExecutionTestWorkflow TestTriggerExecution = "testworkflow"
)

// TestTriggerConcurrencyPolicy defines concurrency policy for test triggers
// +kubebuilder:validation:Enum=allow;forbid;replace
type TestTriggerConcurrencyPolicy string

// List of TestTriggerConcurrencyPolicy
const (
	TestTriggerConcurrencyPolicyAllow   TestTriggerConcurrencyPolicy = "allow"
	TestTriggerConcurrencyPolicyForbid  TestTriggerConcurrencyPolicy = "forbid"
	TestTriggerConcurrencyPolicyReplace TestTriggerConcurrencyPolicy = "replace"
)

// TestTriggerSelector is used for selecting Kubernetes Objects
type TestTriggerSelector struct {
	// Name selector is used to identify a Kubernetes Object based on the metadata name
	Name string `json:"name,omitempty"`
	// kubernetes resource name regex
	NameRegex string `json:"nameRegex,omitempty"`
	// Namespace of the Kubernetes object
	Namespace string `json:"namespace,omitempty"`
	// kubernetes resource namespace regex
	NamespaceRegex string `json:"namespaceRegex,omitempty"`
	// LabelSelector is used to identify a group of Kubernetes Objects based on their metadata labels
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// TestTriggerStatus defines the observed state of TestTrigger
type TestTriggerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// TestTriggerConditionStatuses defines condition statuses for test triggers
// +kubebuilder:validation:Enum=True;False;Unknown
type TestTriggerConditionStatuses string

// List of TestTriggerConditionStatuses
const (
	TRUE_TestTriggerConditionStatuses    TestTriggerConditionStatuses = "True"
	FALSE_TestTriggerConditionStatuses   TestTriggerConditionStatuses = "False"
	UNKNOWN_TestTriggerConditionStatuses TestTriggerConditionStatuses = "Unknown"
)

// TestTriggerCondition is used for definition of the condition for test triggers
type TestTriggerCondition struct {
	Status *TestTriggerConditionStatuses `json:"status"`
	// test trigger condition
	Type_ string `json:"type"`
	// test trigger condition reason
	Reason string `json:"reason,omitempty"`
	// duration in seconds in the past from current time when the condition is still valid
	Ttl int32 `json:"ttl,omitempty"`
}

// TestTriggerConditionSpec defines the condition specification for TestTrigger
type TestTriggerConditionSpec struct {
	// list of test trigger conditions
	Conditions []TestTriggerCondition `json:"conditions,omitempty"`
	// duration in seconds the test trigger waits for conditions, until its stopped
	Timeout int32 `json:"timeout,omitempty"`
	// duration in seconds the test trigger waits between condition check
	Delay int32 `json:"delay,omitempty"`
}

// TestTriggerProbe is used for definition of the probe for test triggers
type TestTriggerProbe struct {
	// test trigger condition probe scheme to connect to host, default is http
	Scheme string `json:"scheme,omitempty"`
	// test trigger condition probe host, default is pod ip or service name
	Host string `json:"host,omitempty"`
	// test trigger condition probe path to check, default is /
	Path string `json:"path,omitempty"`
	// test trigger condition probe port to connect
	Port int32 `json:"port,omitempty"`
	// test trigger condition probe headers to submit
	Headers map[string]string `json:"headers,omitempty"`
}

// TestTriggerProbeSpec defines the probe specification for TestTrigger
type TestTriggerProbeSpec struct {
	// list of test trigger probes
	Probes []TestTriggerProbe `json:"probes,omitempty"`
	// duration in seconds the test trigger waits for probes, until its stopped
	Timeout int32 `json:"timeout,omitempty"`
	// duration in seconds the test trigger waits between probes
	Delay int32 `json:"delay,omitempty"`
}

// supported action parameters for test triggers
type TestTriggerActionParameters struct {
	// configuration to pass for the workflow
	Config map[string]string `json:"config,omitempty"`
	// test workflow execution tags
	Tags map[string]string `json:"tags,omitempty"`
	// Target helps decide on which runner the execution is scheduled.
	Target *commonv1.Target `json:"target,omitempty" expr:"include"`
}

// TestTriggerContentSelector defines what content to watch for trigger events
type TestTriggerContentSelector struct {
	// Git specifies a git repository to watch for changes
	Git *TestTriggerContentGitSpec `json:"git,omitempty"`
}

// TestTriggerContentGitSpec defines git repository configuration for content triggers
type TestTriggerContentGitSpec struct {
	// URI of the git repository
	Uri string `json:"uri"`
	// Branches is a list of branch name patterns to watch (glob supported, e.g. "main", "release/*").
	// If empty, all branches are watched.
	Branches []string `json:"branches,omitempty"`
	// BranchesIgnore is a list of branch name patterns to exclude (glob supported).
	// Takes precedence over Branches when both match.
	BranchesIgnore []string `json:"branchesIgnore,omitempty"`
	// Paths is a list of file/directory paths to watch for changes (glob supported).
	// If empty, all paths are watched.
	Paths []string `json:"paths,omitempty"`
	// PathsIgnore is a list of file/directory path patterns to exclude (glob supported).
	// Takes precedence over Paths when both match.
	PathsIgnore []string `json:"pathsIgnore,omitempty"`
	// Tags is a list of tag name patterns to watch (glob supported, e.g. "v*", "v1.*").
	// If empty, tag events are not watched unless Branches is also empty (watch all).
	Tags []string `json:"tags,omitempty"`
	// TagsIgnore is a list of tag name patterns to exclude (glob supported).
	// Takes precedence over Tags when both match.
	TagsIgnore []string `json:"tagsIgnore,omitempty"`
	// Plain text username to fetch with
	Username string `json:"username,omitempty"`
	// External username to fetch with
	UsernameFrom *corev1.EnvVarSource `json:"usernameFrom,omitempty"`
	// Plain text token to fetch with.
	// Warning: this stores sensitive credentials directly in the CRD spec, which may be persisted in etcd
	// and exposed through Kubernetes API reads or logs. Prefer TokenFrom instead.
	Token string `json:"token,omitempty"`
	// External token to fetch with. Preferred for sensitive credentials.
	TokenFrom *corev1.EnvVarSource `json:"tokenFrom,omitempty"`
	// Plain text SSH private key to fetch with.
	// Warning: this stores sensitive credentials directly in the CRD spec, which may be persisted in etcd
	// and exposed through Kubernetes API reads or logs. Prefer SshKeyFrom instead.
	SshKey string `json:"sshKey,omitempty"`
	// External SSH private key to fetch with. Preferred for sensitive credentials.
	SshKeyFrom *corev1.EnvVarSource `json:"sshKeyFrom,omitempty"`
	// Authorization type for the credentials
	AuthType testsv3.GitAuthType `json:"authType,omitempty"`
	// PullRequest specifies pull request trigger configuration.
	// When set, the informer uses the GitHub API to poll for PR events.
	PullRequest *TestTriggerContentGitPullRequest `json:"pullRequest,omitempty"`
}

// TestTriggerContentGitPullRequest defines pull request trigger configuration.
type TestTriggerContentGitPullRequest struct {
	// Types is a list of PR activity types to watch (e.g. "opened", "synchronize", "reopened", "closed").
	// If empty, all types are watched.
	Types []string `json:"types,omitempty"`
	// Branches is a list of base branch name patterns to watch (glob supported).
	// If empty, PRs targeting any base branch are watched.
	Branches []string `json:"branches,omitempty"`
	// BranchesIgnore is a list of base branch name patterns to exclude (glob supported).
	// Takes precedence over Branches when both match.
	BranchesIgnore []string `json:"branchesIgnore,omitempty"`
}

//+kubebuilder:object:root=true

// TestTriggerList contains a list of TestTrigger
type TestTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestTrigger `json:"items"`
}
