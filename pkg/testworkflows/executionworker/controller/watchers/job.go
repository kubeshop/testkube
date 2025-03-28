package watchers

import (
	"encoding/json"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type job struct {
	original *batchv1.Job
}

type Job interface {
	Original() *batchv1.Job
	Name() string
	Namespace() string
	ResourceId() string
	RootResourceId() string
	RunnerId() string
	CreationTimestamp() time.Time
	FinishTimestamp() time.Time
	Finished() bool
	ActionGroups() (actiontypes.ActionGroups, error)
	Signature() ([]stage.Signature, error)
	InternalConfig() (testworkflowconfig.InternalConfig, error)
	ScheduledAt() (time.Time, error)
	ExecutionError() string
	Debug() string
}

func NewJob(original *batchv1.Job) Job {
	return &job{original: original}
}

func (j *job) Original() *batchv1.Job {
	return j.original
}

func (j *job) Name() string {
	return j.original.Name
}

func (j *job) Namespace() string {
	return j.original.Namespace
}

func (j *job) ResourceId() string {
	return j.original.Spec.Template.Labels[constants.ResourceIdLabelName]
}

func (j *job) RootResourceId() string {
	return j.original.Spec.Template.Labels[constants.RootResourceIdLabelName]
}

func (j *job) RunnerId() string {
	return j.original.Spec.Template.Labels[constants.RunnerIdLabelName]
}

func (j *job) CreationTimestamp() time.Time {
	return j.original.CreationTimestamp.Time
}

func (j *job) FinishTimestamp() time.Time {
	if !j.Finished() {
		return time.Time{}
	}
	return GetJobCompletionTimestamp(j.original)
}

func (j *job) Finished() bool {
	return IsJobFinished(j.original)
}

func (j *job) ActionGroups() (actions actiontypes.ActionGroups, err error) {
	err = json.Unmarshal([]byte(j.original.Spec.Template.Annotations[constants.SpecAnnotationName]), &actions)
	return
}

func (j *job) InternalConfig() (cfg testworkflowconfig.InternalConfig, err error) {
	err = json.Unmarshal([]byte(j.original.Spec.Template.Annotations[constants.InternalAnnotationName]), &cfg)
	return
}

func (j *job) Signature() ([]stage.Signature, error) {
	return stage.GetSignatureFromJSON([]byte(j.original.Spec.Template.Annotations[constants.SignatureAnnotationName]))
}

func (j *job) ScheduledAt() (time.Time, error) {
	return time.Parse(time.RFC3339Nano, j.original.Spec.Template.Annotations[constants.ScheduledAtAnnotationName])
}

func (j *job) ExecutionError() string {
	return GetJobError(j.original)
}

func (j *job) Debug() string {
	if j.original == nil {
		return "unknown"
	}
	state := "found"
	if j.original.Status.Active > 0 {
		state += fmt.Sprintf(", active: %d", j.original.Status.Active)
	}
	if j.original.Status.Failed > 0 {
		state += fmt.Sprintf(", failed: %d", j.original.Status.Failed)
	}
	if j.original.Status.Succeeded > 0 {
		state += fmt.Sprintf(", succeeded: %d", j.original.Status.Succeeded)
	}
	if j.original.Status.Ready != nil {
		state += fmt.Sprintf(", ready: %d", *j.original.Status.Ready)
	}
	if j.original.Status.Terminating != nil {
		state += fmt.Sprintf(", terminating: %d", *j.original.Status.Terminating)
	}
	if j.original.Status.UncountedTerminatedPods != nil {
		state += fmt.Sprintf(", uncounted terminated pods: (failed) %d (succeeded) %d",
			len(j.original.Status.UncountedTerminatedPods.Failed),
			len(j.original.Status.UncountedTerminatedPods.Succeeded))
	}
	if j.original.Status.StartTime != nil {
		state += ", started"
	}
	if j.original.DeletionTimestamp != nil {
		state += ", deleted"
	}
	if j.original.Status.CompletionTime != nil {
		state += ", completed"
	}
	for i := range j.original.Status.Conditions {
		state += fmt.Sprintf(", %s='%s'", j.original.Status.Conditions[i].Type, j.original.Status.Conditions[i].Status)
	}
	if j.original.Spec.TTLSecondsAfterFinished != nil {
		state += fmt.Sprintf(", ttl after: %ds", *j.original.Spec.TTLSecondsAfterFinished)
	}
	return state
}
