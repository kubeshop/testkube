package v1

type TestWorkflowSpecBase struct {
	// Important: Run "make" to regenerate code after modifying this file

	// events triggering execution of the test workflow
	Events []Event `json:"events,omitempty" expr:"include"`

	// system configuration to define the orchestration behavior
	System *TestWorkflowSystem `json:"system,omitempty" expr:"include"`

	// make the instance configurable with some input data for scheduling it
	Config map[string]ParameterSchema `json:"config,omitempty" expr:"include"`

	// global content that should be fetched into all containers
	Content *Content `json:"content,omitempty" expr:"include"`

	// defaults for the containers for all the TestWorkflow steps
	Container *ContainerConfig `json:"container,omitempty" expr:"include"`

	// configuration for the scheduled job
	Job *JobConfig `json:"job,omitempty" expr:"include"`

	// configuration for the scheduled pod
	Pod *PodConfig `json:"pod,omitempty" expr:"include"`

	// configuration for concurrency policy
	Concurrency *ConcurrencyPolicy `json:"concurrency,omitempty" expr:"include"`

	// configuration for notifications
	// Deprecated: field is not used
	Notifications *NotificationsConfig `json:"notifications,omitempty" expr:"include"`

	// values to be used for test workflow execution
	Execution *TestWorkflowTagSchema `json:"execution,omitempty" expr:"include"`
}

type TestWorkflowSystem struct {
	// assume all the steps are pure by default
	PureByDefault *bool `json:"pureByDefault,omitempty"`

	// disable the behavior of merging multiple operations in a single container
	IsolatedContainers *bool `json:"isolatedContainers,omitempty"`
}

// ConcurrencyPolicy defines a policy for running and queueing concurrent executions.
type ConcurrencyPolicy struct {
	// Group ongoing executions by this identifier instead of by workflow name.
	// Use the group identifier if you want the control concurrency across workflows
	Group string `json:"group,omitempty" expr:"include"`

	// The maximum amount of concurrent executions for this workflow or group.
	// The scheduler will check the amount of ongoing executions for this workflow or group and only
	// schedule this workflow when the amount is below its given maximum. When using a group identifier, it is
	// recommended to keep the maximum in sync through a WorkflowTemplate.
	Max int `json:"max,omitempty" expr:"include"`

	// Whether the oldest in progress execution should be cancelled to be replaced with the latest queued one.
	CancelInProgress bool `json:"cancelInProgress,omitempty" expr:"include"`
}
