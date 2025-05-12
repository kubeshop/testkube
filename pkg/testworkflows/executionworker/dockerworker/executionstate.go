package dockerworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"

	constants2 "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

var (
	ErrMissingData = errors.New("missing data to fulfill request")
)

type executionState struct {
	options    *ExecutionStateOptions
	containers []types.ContainerJSON
}

type ExecutionStateOptions struct {
	ResourceId     string
	RootResourceId string
	Namespace      string
	Signature      []stage.Signature
	ActionGroups   actiontypes.ActionGroups
	ScheduledAt    time.Time
}

func NewExecutionState(containers []types.ContainerJSON, opts *ExecutionStateOptions) *executionState {
	if opts == nil {
		opts = &ExecutionStateOptions{}
	}
	return &executionState{
		options:    opts,
		containers: containers,
	}
}

func (e *executionState) Available() bool {
	return len(e.containers) > 0
}

func (e *executionState) Scheduled() bool {
	return e.Available()
}

func (e *executionState) ContainersReady() bool {
	sig, err := e.Signature()
	if err != nil {
		return false
	}
	sigSequence := stage.MapSignatureToSequence(sig)
	return len(sigSequence) == len(e.containers) // TODO: or just one?
}

func (e *executionState) CompletionTimestamp() time.Time {
	failed := false
	finished := 0
	maxTs := time.Time{}
	for i := range e.containers {
		if e.containers[i].State != nil && e.containers[i].State.FinishedAt != "" {
			ts, _ := time.Parse(time.RFC3339Nano, e.containers[i].State.FinishedAt)
			if !ts.IsZero() {
				finished++
			}
			if ts.After(maxTs) {
				maxTs = ts
			}
		}
		if e.containers[i].State != nil && (e.containers[i].State.ExitCode != 0 || e.containers[i].State.Error != "") {
			failed = true
		}
	}
	if failed || finished == len(e.containers) {
		return maxTs
	}
	return time.Time{}
}

func (e *executionState) container(name string) *types.ContainerJSON {
	name = fmt.Sprintf("/%s-%s", e.options.ResourceId, name)
	for i := range e.containers {
		if e.containers[i].Name == name {
			return &e.containers[i]
		}
	}
	return nil
}

func (e *executionState) ContainerStartTimestamp(name string) time.Time {
	container := e.container(name)
	if container == nil || container.State == nil {
		return time.Time{}
	}
	ts, _ := time.Parse(time.RFC3339Nano, container.State.StartedAt)
	return ts
}

func (e *executionState) ResourceId() string {
	if e.options.ResourceId != "" {
		return e.options.ResourceId
	}
	if len(e.containers) > 0 {
		return e.containers[0].Config.Labels[constants.ResourceIdLabelName]
	}
	return ""
}

func (e *executionState) RootResourceId() string {
	if len(e.containers) > 0 {
		return e.containers[0].Config.Labels[constants.RootResourceIdLabelName]
	}
	return ""
}

func (e *executionState) RunnerId() string {
	if len(e.containers) > 0 {
		return e.containers[0].Config.Labels[constants.RunnerIdLabelName]
	}
	return ""
}

func (e *executionState) Events() watchers.Events {
	return watchers.NewEvents(nil) // FIXME
}

func (e *executionState) Signature() ([]stage.Signature, error) {
	if len(e.options.Signature) > 0 {
		return e.options.Signature, nil
	}
	if len(e.containers) > 0 {
		return stage.GetSignatureFromJSON([]byte(e.containers[0].Config.Labels[constants.SignatureAnnotationName]))
	}
	return nil, ErrMissingData
}

func (e *executionState) InternalConfig() (cfg testworkflowconfig.InternalConfig, err error) {
	if len(e.containers) == 0 {
		return cfg, ErrMissingData
	}
	err = json.Unmarshal([]byte(e.containers[0].Config.Labels[constants.InternalAnnotationName]), &cfg)
	return
}

func (e *executionState) ScheduledAt() time.Time {
	for i := range e.containers {
		ts, _ := time.Parse(time.RFC3339Nano, e.containers[i].Config.Labels[constants.ScheduledAtAnnotationName])
		if !ts.IsZero() {
			return ts
		}
	}
	return e.options.ScheduledAt
}

func (e *executionState) ActionGroups() (actions actiontypes.ActionGroups, err error) {
	if e.options.ActionGroups != nil {
		return e.options.ActionGroups, nil
	}
	if len(e.containers) == 0 {
		return nil, ErrMissingData
	}
	err = json.Unmarshal([]byte(e.containers[0].Config.Labels[constants.SpecAnnotationName]), &actions)
	return
}

func (e *executionState) ContainerStarted(name string) bool {
	container := e.container(name)
	if container == nil || container.State == nil {
		return false
	}
	return container.State.Running
}

func (e *executionState) ContainerFinished(name string) bool {
	container := e.container(name)
	if container == nil || container.State == nil {
		return false
	}
	ts, _ := time.Parse(time.RFC3339Nano, container.State.FinishedAt)
	return !ts.IsZero()
}

func (e *executionState) ContainerFailed(name string) bool {
	container := e.container(name)
	if container == nil || container.State == nil {
		return false
	}
	return container.State.ExitCode != 0 || container.State.Error != ""
}

func (e *executionState) ContainerResult(name, executionError string) watchers.ContainerResult {
	// FIXME
	return watchers.ContainerResult{}
}

func (e *executionState) EstimatedJobCreationTimestamp() time.Time {
	ts := time.Time{}
	for i := range e.containers {
		cts, _ := time.Parse(time.RFC3339Nano, e.containers[i].Created)
		if ts.IsZero() || cts.Before(ts) {
			ts = cts
		}
	}
	if ts.IsZero() {
		return e.ScheduledAt()
	}
	return ts
}

func (e *executionState) EstimatedPodCreationTimestamp() time.Time {
	ts := time.Time{}
	for i := range e.containers {
		cts, _ := time.Parse(time.RFC3339Nano, e.containers[i].Created)
		if ts.IsZero() || cts.After(ts) {
			ts = cts
		}
	}
	if ts.IsZero() {
		return e.ScheduledAt()
	}
	return ts
}

func (e *executionState) EstimatedPodStartTimestamp() time.Time {
	ts := time.Time{}
	for i := range e.containers {
		if e.containers[i].State != nil && e.containers[i].State.StartedAt != "" {
			cts, _ := time.Parse(time.RFC3339Nano, e.containers[i].State.StartedAt)
			if ts.IsZero() || cts.Before(ts) {
				ts = cts
			}
		}
	}
	return ts
}

func (e *executionState) PodStarted() bool {
	return !e.EstimatedPodStartTimestamp().IsZero()
}

func (e *executionState) Completed() bool {
	return !e.CompletionTimestamp().IsZero()
}

func (e *executionState) ExecutionError() string {
	for i := range e.containers {
		name := fmt.Sprintf("%d", i+1)
		container := e.container(name)
		if container.State == nil {
			continue
		}
		if container.State.Error != "" {
			return container.State.Error
		}
		if container.State.ExitCode != 0 {
			switch container.State.ExitCode {
			case int(constants2.CodeAborted):
				return "Aborted"
			case int(constants2.CodeInputError):
				return "Invalid Input"
			case int(constants2.CodeInternal):
				return "Fatal Error"
			default:
				return "Unknown Error"
			}
		}
	}
	return ""
}

func (e *executionState) Debug() map[string]string {
	result := map[string]string{"containers": "unknown"} // FIXME
	return result
}
