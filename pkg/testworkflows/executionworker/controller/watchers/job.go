package watchers

import (
	"encoding/json"
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
