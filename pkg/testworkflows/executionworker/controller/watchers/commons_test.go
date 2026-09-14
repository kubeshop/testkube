package watchers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetJobError(tt.job))
		})
	}
}
