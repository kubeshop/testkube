package watchers

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"

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
	ContainerStarted(name string) bool
	ContainerFinished(name string) bool
	ContainerFinishTimestamp(name string) time.Time
	ContainerResult(name string, executionError string) ContainerResult
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
	nodeName := p.original.Status.NominatedNodeName
	if nodeName == "" {
		nodeName = p.original.Spec.NodeName
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

func (p *pod) ContainerStarted(name string) bool {
	return IsContainerStarted(p.original, name)
}

func (p *pod) ContainerFinished(name string) bool {
	return IsContainerFinished(p.original, name)
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

func (p *pod) ExecutionError() string {
	return GetPodError(p.original)
}