package watchers

import (
	"encoding/json"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type pod struct {
	original *corev1.Pod
}

type Pod interface {
	Original() *corev1.Pod
	Name() string
	Namespace() string
	ResourceId() string
	RootResourceId() string
	NodeName() string
	IP() string
	CreationTimestamp() time.Time
	StartTimestamp() time.Time
	FinishTimestamp() time.Time
	Finished() bool
	ActionGroups() (actiontypes.ActionGroups, error)
	Signature() ([]stage.Signature, error)
	InternalConfig() (testworkflowconfig.InternalConfig, error)
	ScheduledAt() (time.Time, error)
	ContainerStarted(name string) bool
	ContainerFinished(name string) bool
	ContainerFailed(name string) bool
	ContainerStartTimestamp(name string) time.Time
	ContainerFinishTimestamp(name string) time.Time
	ContainerResult(name string, executionError string) ContainerResult
	ContainersReady() bool
	ContainerError() string
	ExecutionError() string
}

func NewPod(original *corev1.Pod) Pod {
	return &pod{original: original}
}

func (p *pod) Original() *corev1.Pod {
	return p.original
}

func (p *pod) Name() string {
	return p.original.Name
}

func (p *pod) Namespace() string {
	return p.original.Namespace
}

func (p *pod) ResourceId() string {
	return p.original.Labels[constants.ResourceIdLabelName]
}

func (p *pod) RootResourceId() string {
	return p.original.Labels[constants.RootResourceIdLabelName]
}

func (p *pod) NodeName() string {
	nodeName := p.original.Spec.NodeName
	if nodeName == "" {
		nodeName = p.original.Status.NominatedNodeName
	}
	return nodeName
}

func (p *pod) IP() string {
	return p.original.Status.PodIP
}

func (p *pod) CreationTimestamp() time.Time {
	return p.original.CreationTimestamp.Time
}

func (p *pod) StartTimestamp() time.Time {
	if !p.original.Status.StartTime.IsZero() {
		return p.original.Status.StartTime.Time
	}
	status := GetContainerStatus(p.original, "1")
	if status != nil {
		if status.State.Running != nil {
			return status.State.Running.StartedAt.Time
		} else if status.State.Terminated != nil {
			return status.State.Terminated.StartedAt.Time
		}
	}
	return time.Time{}
}

func (p *pod) FinishTimestamp() time.Time {
	if !p.Finished() {
		return time.Time{}
	}
	return GetPodCompletionTimestamp(p.original)
}

func (p *pod) Finished() bool {
	return IsPodFinished(p.original)
}

func (p *pod) ActionGroups() (actions actiontypes.ActionGroups, err error) {
	err = json.Unmarshal([]byte(p.original.Annotations[constants.SpecAnnotationName]), &actions)
	return
}

func (p *pod) Signature() ([]stage.Signature, error) {
	return stage.GetSignatureFromJSON([]byte(p.original.Annotations[constants.SignatureAnnotationName]))
}

func (p *pod) InternalConfig() (cfg testworkflowconfig.InternalConfig, err error) {
	err = json.Unmarshal([]byte(p.original.Annotations[constants.InternalAnnotationName]), &cfg)
	return
}

func (p *pod) ScheduledAt() (time.Time, error) {
	return time.Parse(time.RFC3339Nano, p.original.Annotations[constants.ScheduledAtAnnotationName])
}

func (p *pod) ContainerStarted(name string) bool {
	return IsContainerStarted(p.original, name)
}

func (p *pod) ContainerFinished(name string) bool {
	return IsContainerFinished(p.original, name)
}

func (p *pod) ContainerFailed(name string) bool {
	status := GetContainerStatus(p.original, name)
	if status == nil {
		return false
	}
	return status.State.Terminated != nil && status.State.Terminated.Reason != "" && status.State.Terminated.Reason != "Completed"
}

func (p *pod) ContainerStartTimestamp(name string) time.Time {
	status := GetContainerStatus(p.original, name)
	if status == nil {
		return time.Time{}
	}
	if status.State.Running != nil {
		return status.State.Running.StartedAt.Time
	} else if status.State.Terminated != nil {
		return status.State.Terminated.StartedAt.Time
	}
	return time.Time{}
}

func (p *pod) ContainerFinishTimestamp(name string) time.Time {
	status := GetContainerStatus(p.original, name)
	if status == nil || status.State.Terminated == nil {
		return time.Time{}
	}
	return status.State.Terminated.FinishedAt.Time
}

func (p *pod) ContainerResult(name string, executionError string) ContainerResult {
	podExecutionError := p.ExecutionError()
	if podExecutionError != "" {
		executionError = podExecutionError
	}
	return ReadContainerResult(GetContainerStatus(p.original, name), executionError)
}

func (p *pod) ContainerError() string {
	// Iterate over all init containers, as even unrelated to Test Workflows will cause error
	for _, c := range p.original.Status.InitContainerStatuses {
		if c.State.Terminated != nil && c.State.Terminated.Reason != "" && c.State.Terminated.Reason != "Completed" {
			return c.State.Terminated.Reason
		}
	}

	// Check only for the last Test Workflow's container, as this is the one we are interested in
	for _, c := range p.original.Status.ContainerStatuses {
		// Check for the container that has number in it, as it's likely TestWorkflow's one
		if _, err := strconv.ParseInt(c.Name, 10, 64); err != nil {
			continue
		}
		if c.State.Terminated != nil && c.State.Terminated.Reason != "" && c.State.Terminated.Reason != "Completed" {
			return c.State.Terminated.Reason
		}
	}

	return ""
}

func (p *pod) ContainersReady() bool {
	// Check for the init containers (active one needs to be ready)
	for _, c := range p.original.Spec.InitContainers {
		if c.ReadinessProbe != nil {
			status := GetContainerStatus(p.original, c.Name)
			if status == nil {
				return false
			} else if status.State.Running != nil || status.State.Waiting != nil {
				return status.Ready
			}
		}
	}

	// Check for the actual containers (all with the readiness probe needs to be ready)
	for _, c := range p.original.Spec.Containers {
		if c.ReadinessProbe != nil {
			status := GetContainerStatus(p.original, c.Name)
			if status == nil || !status.Ready {
				return false
			}
		}
	}
	return true
}

func (p *pod) ExecutionError() string {
	errStr := GetPodError(p.original)
	if errStr == "Error" || errStr == "" {
		return p.ContainerError()
	}
	return errStr
}
