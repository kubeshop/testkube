package watchers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

// jobWithCondition returns a job with a deadline of 60 seconds and one true condition.
func jobWithCondition(conditionType batchv1.JobConditionType, reason string) *batchv1.Job {
	return &batchv1.Job{
		Spec: batchv1.JobSpec{ActiveDeadlineSeconds: common.Ptr(int64(60))},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{
			Type:   conditionType,
			Status: corev1.ConditionTrue,
			Reason: reason,
		}}},
	}
}

func TestGetJobError(t *testing.T) {
	deleted := metav1.NewTime(time.Now())
	tests := []struct {
		name string
		job  *batchv1.Job
		want string
	}{
		{
			name: "actor and reason token",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationActor:  string(testkube.StopActorRunner),
				constants.AnnotationTerminationReason: string(testkube.StopReasonExecutionStuck),
			}}},
			want: "by the runner: the execution is stuck in the running state",
		},
		{
			name: "actor, reason token, and detail",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationActor:  string(testkube.StopActorRunner),
				constants.AnnotationTerminationReason: string(testkube.StopReasonWorkerResumeFailed),
				constants.AnnotationTerminationDetail: "pod not found",
			}}},
			want: "by the runner: the parallel worker could not be resumed: pod not found",
		},
		{
			name: "actor without a reason",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationActor: string(testkube.StopActorUser),
			}}},
			want: "by the user",
		},
		{
			name: "token without words yet keeps its raw text",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationActor:  string(testkube.StopActorControlPlane),
				constants.AnnotationTerminationReason: "later-added",
			}}},
			want: "by the control plane: later-added",
		},
		{
			name: "free text without an actor, as an older worker writes it",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationReason: "Job has been aborted by the system",
			}}},
			want: "Job has been aborted by the system",
		},
		{
			name: "deleted job with no annotation",
			job:  &batchv1.Job{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &deleted}},
			want: "by the system",
		},
		{
			name: "job that still runs",
			job:  &batchv1.Job{},
			want: "",
		},
		{
			name: "job that failed on its deadline",
			job:  jobWithCondition(batchv1.JobFailed, ReasonDeadlineExceeded),
			want: "Job timed out after 60 seconds",
		},
		{
			name: "a deadline of one second uses the singular",
			job: func() *batchv1.Job {
				job := jobWithCondition(batchv1.JobFailed, ReasonDeadlineExceeded)
				job.Spec.ActiveDeadlineSeconds = common.Ptr(int64(1))
				return job
			}(),
			want: "Job timed out after 1 second",
		},
		{
			name: "job that is about to fail on its deadline",
			job:  jobWithCondition(batchv1.JobFailureTarget, ReasonDeadlineExceeded),
			want: "Job timed out after 60 seconds",
		},
		{
			name: "job that is about to fail for another reason",
			job:  jobWithCondition(batchv1.JobFailureTarget, "BackoffLimitExceeded"),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetJobError(tt.job))
		})
	}
}

func TestGetJobStop(t *testing.T) {
	deleted := metav1.NewTime(time.Now())
	aborted := string(testkube.ABORTED_TestWorkflowStatus)
	tests := []struct {
		name string
		job  *batchv1.Job
		want testkube.Stop
	}{
		{
			name: "reads the four annotations",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationCode:   string(testkube.CANCELED_TestWorkflowStatus),
				constants.AnnotationTerminationActor:  string(testkube.StopActorUser),
				constants.AnnotationTerminationReason: string(testkube.StopReasonAbortAll),
				constants.AnnotationTerminationDetail: "the whole workflow",
			}}},
			want: testkube.Stop{
				Code:   string(testkube.CANCELED_TestWorkflowStatus),
				Actor:  testkube.StopActorUser,
				Reason: testkube.StopReasonAbortAll,
				Detail: "the whole workflow",
			},
		},
		{
			name: "names the system for a deleted job without annotations",
			job:  &batchv1.Job{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &deleted}},
			want: testkube.Stop{Code: aborted, Actor: testkube.StopActorSystem},
		},
		{
			name: "keeps the annotated actor of a deleted job",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
				DeletionTimestamp: &deleted,
				Annotations:       map[string]string{constants.AnnotationTerminationActor: string(testkube.StopActorTrigger)},
			}},
			want: testkube.Stop{Code: aborted, Actor: testkube.StopActorTrigger},
		},
		{
			name: "reports the code only for a job that still runs",
			job:  &batchv1.Job{},
			want: testkube.Stop{Code: aborted},
		},
		{
			name: "reports the default code without a job",
			want: testkube.Stop{Code: aborted},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetJobStop(tt.job))
		})
	}
}
