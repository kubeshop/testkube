package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
)

type RetryPolicy struct {
	// how many times at most it should retry
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Count int32 `json:"count,omitempty"`

	// until when it should retry (defaults to: "passed")
	Until string `json:"until,omitempty" expr:"expression"`
}

type StepMeta struct {
	// readable name for the step
	Name string `json:"name,omitempty" expr:"template"`

	// expression to declare under which conditions the step should be run
	// defaults to: "passed", except artifacts where it defaults to "always"
	Condition string `json:"condition,omitempty" expr:"expression"`

	// mark the step as pure, applying optimizations to merge the containers together
	Pure *bool `json:"pure,omitempty"`
}

type StepSource struct {
	// content that should be fetched for this step
	Content *Content `json:"content,omitempty" expr:"include"`
}

type StepDefaults struct {
	// defaults for the containers in this step
	Container *ContainerConfig `json:"container,omitempty" expr:"include"`

	// working directory to use for this step
	WorkingDir *string `json:"workingDir,omitempty" expr:"template"`
}

type StepControl struct {
	// is the step expected to fail
	Negative bool `json:"negative,omitempty"`

	// is the step optional, so its failure won't affect the TestWorkflow result
	Optional bool `json:"optional,omitempty"`

	// pause the step initially
	Paused bool `json:"paused,omitempty" expr:"ignore"`

	// policy for retrying the step
	Retry *RetryPolicy `json:"retry,omitempty" expr:"include"`

	// maximum time this step may take
	Timeout string `json:"timeout,omitempty" expr:"template"`
}

type StepOperations struct {
	// delay before the step
	// +kubebuilder:validation:Pattern=^((0|[1-9][0-9]*)h)?((0|[1-9][0-9]*)m)?((0|[1-9][0-9]*)s)?((0|[1-9][0-9]*)ms)?$
	Delay string `json:"delay,omitempty"`

	// script to run in a default shell for the container
	Shell string `json:"shell,omitempty" expr:"template"`

	// run specific container in the current step
	Run *StepRun `json:"run,omitempty" expr:"include"`

	// execute other Testkube resources
	Execute *StepExecute `json:"execute,omitempty" expr:"include"`

	// scrape artifacts from the volumes
	Artifacts *StepArtifacts `json:"artifacts,omitempty" expr:"include"`
}

type IndependentStep struct {
	StepMeta    `json:",inline" expr:"include"`
	StepControl `json:",inline" expr:"include"`
	StepSource  `json:",inline" expr:"include"`

	// list of accompanying services to start
	Services map[string]IndependentServiceSpec `json:"services,omitempty" expr:"template,include"`

	StepDefaults `json:",inline" expr:"include"`

	// steps to run before other operations in this step
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Setup []IndependentStep `json:"setup,omitempty" expr:"include"`

	StepOperations `json:",inline" expr:"include"`

	// instructions for parallel execution
	Parallel *IndependentStepParallel `json:"parallel,omitempty" expr:"include"`

	// sub-steps to run
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Steps []IndependentStep `json:"steps,omitempty" expr:"include"`
}

type Step struct {
	StepMeta    `json:",inline" expr:"include"`
	StepControl `json:",inline" expr:"include"`

	// multiple templates to include in this step
	Use []TemplateRef `json:"use,omitempty" expr:"include"`

	StepSource `json:",inline" expr:"include"`

	// list of accompanying services to start
	Services map[string]ServiceSpec `json:"services,omitempty" expr:"template,include"`

	StepDefaults `json:",inline" expr:"include"`

	// steps to run before other operations in this step
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Setup []Step `json:"setup,omitempty" expr:"include"`

	StepOperations `json:",inline" expr:"include"`

	// single template to run in this step
	Template *TemplateRef `json:"template,omitempty" expr:"include"`

	// instructions for parallel execution
	Parallel *StepParallel `json:"parallel,omitempty" expr:"include"`

	// sub-steps to run
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Steps []Step `json:"steps,omitempty" expr:"include"`
}

type StepRun struct {
	ContainerConfig `json:",inline" expr:"include"`

	// script to run in a default shell for the container
	Shell *string `json:"shell,omitempty" expr:"template"`
}

type StepExecute struct {
	// how many resources could be scheduled in parallel
	Parallelism int32 `json:"parallelism,omitempty"`

	// only schedule the resources, don't watch the results (unless it is needed for parallelism)
	Async bool `json:"async,omitempty"`

	// tests to run
	Tests []StepExecuteTest `json:"tests,omitempty" expr:"include"`

	// workflows to run
	Workflows []StepExecuteWorkflow `json:"workflows,omitempty" expr:"include"`
}

type TarballRequest struct {
	// path to load the files from
	From string `json:"from,omitempty" expr:"template"`

	// file patterns to pack
	Files *DynamicList `json:"files,omitempty" expr:"template"`
}

type StepExecuteStrategy struct {
	// matrix of parameters to spawn instances (static)
	Matrix map[string]DynamicList `json:"matrix,omitempty" expr:"force"`

	// static number of sharded instances to spawn
	Count *intstr.IntOrString `json:"count,omitempty" expr:"expression"`

	// dynamic number of sharded instances to spawn - it will be lowered if there is not enough sharded values
	MaxCount *intstr.IntOrString `json:"maxCount,omitempty" expr:"expression"`

	// parameters that should be distributed across sharded instances
	Shards map[string]DynamicList `json:"shards,omitempty" expr:"force"`
}

type StepExecuteTest struct {
	// test name to run
	Name string `json:"name,omitempty" expr:"template"`

	// test execution description to display
	Description string `json:"description,omitempty" expr:"template"`

	StepExecuteStrategy `json:",inline" expr:"include"`

	// pack some data from the original file system to serve them down
	Tarball map[string]TarballRequest `json:"tarball,omitempty" expr:"template,include"`

	// pass the execution request overrides
	ExecutionRequest *TestExecutionRequest `json:"executionRequest,omitempty" expr:"include"`
}

type StepExecuteWorkflow struct {
	// workflow name to run
	Name string `json:"name,omitempty" expr:"template"`

	// selector is used to identify a group of test workflows based on their metadata labels
	Selector *metav1.LabelSelector `json:"selector,omitempty" expr:"include"`

	// test workflow execution description to display
	Description string `json:"description,omitempty" expr:"template"`

	StepExecuteStrategy `json:",inline" expr:"include"`

	// unique execution name to use
	ExecutionName string `json:"executionName,omitempty" expr:"template"`

	// pack some data from the original file system to serve them down
	Tarball map[string]TarballRequest `json:"tarball,omitempty" expr:"template,include"`

	// configuration to pass for the workflow
	Config map[string]intstr.IntOrString `json:"config,omitempty" expr:"template"`

	// Targets helps decide on which runner the execution is scheduled.
	Target *commonv1.Target `json:"target,omitempty" expr:"include"`
}

type StepParallel struct {
	// how many resources could be scheduled in parallel
	Parallelism int32 `json:"parallelism,omitempty"`

	// abort remaining parallel workers on first failure
	FailFast bool `json:"failFast,omitempty"`

	StepExecuteStrategy `json:",inline" expr:"include"`

	// worker description to display
	Description string `json:"description,omitempty" expr:"template"`

	// should save logs for the parallel step (true if not specified)
	Logs *string `json:"logs,omitempty" expr:"expression"`

	// instructions for transferring files
	Transfer []StepParallelTransfer `json:"transfer,omitempty" expr:"include"`

	// instructions for fetching files back
	Fetch []StepParallelFetch `json:"fetch,omitempty" expr:"include"`

	StepControl    `json:",inline" expr:"include"`
	StepOperations `json:",inline" expr:"include"`

	// single template to run in this step
	Template *TemplateRef `json:"template,omitempty" expr:"include"`

	// templates to include at a top-level of workflow
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Use []TemplateRef `json:"use,omitempty" expr:"include"`

	// events triggering execution of the test workflow
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Events []Event `json:"events,omitempty" expr:"include"`

	// system configuration to define the orchestration behavior
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	System *TestWorkflowSystem `json:"system,omitempty" expr:"include"`

	// make the instance configurable with some input data for scheduling it
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Config map[string]ParameterSchema `json:"config,omitempty" expr:"include"`

	// global content that should be fetched into all containers
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Content *Content `json:"content,omitempty" expr:"include"`

	// defaults for the containers for all the TestWorkflow steps
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Container *ContainerConfig `json:"container,omitempty" expr:"include"`

	// configuration for the scheduled job
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Job *JobConfig `json:"job,omitempty" expr:"include"`

	// configuration for the scheduled pod
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Pod *PodConfig `json:"pod,omitempty" expr:"include"`

	// configuration for notifications
	// Deprecated: field is not used
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Notifications *NotificationsConfig `json:"notifications,omitempty" expr:"include"`

	// values to be used for test workflow execution
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Execution *TestWorkflowExecutionSchema `json:"execution,omitempty" expr:"include"`

	// list of accompanying services to start
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Services map[string]ServiceSpec `json:"services,omitempty" expr:"template,include"`

	// steps for setting up the workflow
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Setup []Step `json:"setup,omitempty" expr:"include"`

	// steps to execute in the workflow
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Steps []Step `json:"steps,omitempty" expr:"include"`

	// steps to run at the end of the workflow
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	After []Step `json:"after,omitempty" expr:"include"`

	// list of accompanying permanent volume claims
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Pvcs map[string]corev1.PersistentVolumeClaimSpec `json:"pvcs,omitempty" expr:"template,include"`
}

func (sp StepParallel) NewTestWorkflowSpec() *TestWorkflowSpec {
	return &TestWorkflowSpec{
		Use: sp.Use,
		TestWorkflowSpecBase: TestWorkflowSpecBase{
			Events:        sp.Events,
			System:        sp.System,
			Config:        sp.Config,
			Content:       sp.Content,
			Container:     sp.Container,
			Job:           sp.Job,
			Pod:           sp.Pod,
			Notifications: sp.Notifications,
			Execution:     sp.Execution,
		},
		Services: sp.Services,
		Setup:    sp.Setup,
		Steps:    sp.Steps,
		After:    sp.After,
		Pvcs:     sp.Pvcs,
	}
}

type IndependentStepParallel struct {
	// how many resources could be scheduled in parallel
	Parallelism int32 `json:"parallelism,omitempty"`

	// abort remaining parallel workers on first failure
	FailFast bool `json:"failFast,omitempty"`

	StepExecuteStrategy `json:",inline" expr:"include"`

	// worker description to display
	Description string `json:"description,omitempty" expr:"template"`

	// should save logs for the parallel step (true if not specified)
	Logs *string `json:"logs,omitempty" expr:"expression"`

	// instructions for transferring files
	Transfer []StepParallelTransfer `json:"transfer,omitempty" expr:"include"`

	// instructions for fetching files back
	Fetch []StepParallelFetch `json:"fetch,omitempty" expr:"include"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	TestWorkflowTemplateSpec `json:",inline" expr:"include"`

	StepControl    `json:",inline" expr:"include"`
	StepOperations `json:",inline" expr:"include"`
}

type StepParallelTransfer struct {
	// path to load the files from
	From string `json:"from" expr:"template"`

	// file patterns to pack
	Files *DynamicList `json:"files,omitempty" expr:"template"`

	// path where the tarball should be extracted
	To string `json:"to,omitempty" expr:"template"`

	// should it mount a new volume there
	Mount *bool `json:"mount,omitempty" expr:"ignore"`
}

type StepParallelFetch struct {
	// path to load the files from
	From string `json:"from" expr:"template"`

	// file patterns to pack
	Files *DynamicList `json:"files,omitempty" expr:"template"`

	// path where the tarball should be extracted
	To string `json:"to,omitempty" expr:"template"`
}

type StepArtifacts struct {
	// working directory to override, so it will be used as a base dir
	WorkingDir *string `json:"workingDir,omitempty" expr:"template"`
	// compression options for the artifacts
	Compress *ArtifactCompression `json:"compress,omitempty" expr:"include"`
	// paths to fetch from the container
	Paths []string `json:"paths,omitempty" expr:"template"`
}

type ArtifactCompression struct {
	// artifact name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name" expr:"template"`
}

type TestExecutionRequest struct {
	// test execution custom name
	Name string `json:"name,omitempty" expr:"template"`
	// test execution labels
	ExecutionLabels map[string]string `json:"executionLabels,omitempty" expr:"template,template"`
	// variables file content - need to be in format for particular executor (e.g. postman envs file)
	VariablesFile           string                      `json:"variablesFile,omitempty" expr:"template"`
	IsVariablesFileUploaded bool                        `json:"isVariablesFileUploaded,omitempty" expr:"ignore"`
	Variables               map[string]testsv3.Variable `json:"variables,omitempty" expr:"template,force"`
	// test secret uuid
	TestSecretUUID string `json:"testSecretUUID,omitempty" expr:"template"`
	// additional executor binary arguments
	Args []string `json:"args,omitempty" expr:"template"`
	// usage mode for arguments
	ArgsMode testsv3.ArgsModeType `json:"argsMode,omitempty" expr:"template"`
	// executor binary command
	Command []string `json:"command,omitempty" expr:"template"`
	// container executor image
	Image string `json:"image,omitempty" expr:"template"`
	// container executor image pull secrets
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" expr:"template"`
	// whether to start execution sync or async
	Sync bool `json:"sync,omitempty" expr:"ignore"`
	// http proxy for executor containers
	HttpProxy string `json:"httpProxy,omitempty" expr:"template"`
	// https proxy for executor containers
	HttpsProxy string `json:"httpsProxy,omitempty" expr:"template"`
	// negative test will fail the execution if it is a success and it will succeed if it is a failure
	NegativeTest bool `json:"negativeTest,omitempty" expr:"ignore"`
	// Optional duration in seconds the pod may be active on the node relative to
	// StartTime before the system will actively try to mark it failed and kill associated containers.
	// Value must be a positive integer.
	ActiveDeadlineSeconds int64                    `json:"activeDeadlineSeconds,omitempty" expr:"ignore"`
	ArtifactRequest       *testsv3.ArtifactRequest `json:"artifactRequest,omitempty" expr:"force"`
	// job template extensions
	JobTemplate string `json:"jobTemplate,omitempty" expr:"ignore"`
	// cron job template extensions
	CronJobTemplate string `json:"cronJobTemplate,omitempty" expr:"ignore"`
	// script to run before test execution
	PreRunScript string `json:"preRunScript,omitempty" expr:"template"`
	// script to run after test execution
	PostRunScript string `json:"postRunScript,omitempty" expr:"template"`
	// execute post run script before scraping (prebuilt executor only)
	ExecutePostRunScriptBeforeScraping bool `json:"executePostRunScriptBeforeScraping,omitempty" expr:"ignore"`
	// run scripts using source command (container executor only)
	SourceScripts bool `json:"sourceScripts,omitempty" expr:"ignore"`
	// scraper template extensions
	ScraperTemplate string `json:"scraperTemplate,omitempty" expr:"ignore"`
	// config map references
	EnvConfigMaps []testsv3.EnvReference `json:"envConfigMaps,omitempty" expr:"force"`
	// secret references
	EnvSecrets []testsv3.EnvReference `json:"envSecrets,omitempty" expr:"force"`
	// namespace for test execution (Pro edition only)
	ExecutionNamespace string `json:"executionNamespace,omitempty" expr:"template"`
}
