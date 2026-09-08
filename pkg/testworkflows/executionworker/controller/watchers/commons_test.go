package watchers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
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
			name: "actor and cause",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationActor:  string(executionworkertypes.AbortActorRunner),
				constants.AnnotationTerminationReason: "execution is stuck in running state",
			}}},
			want: "by the runner: execution is stuck in running state",
		},
		{
			name: "actor without a cause",
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				constants.AnnotationTerminationActor: string(executionworkertypes.AbortActorUser),
			}}},
			want: "by the user",
		},
		{
			name: "cause without an actor, as an older worker writes it",
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
