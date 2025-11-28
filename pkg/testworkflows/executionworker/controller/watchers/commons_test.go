package watchers_test

import (
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestGetJobError(t *testing.T) {
	tests := map[string]struct {
		job    *batchv1.Job
		expect string
	}{
		"nil Job": {
			job:    nil,
			expect: "",
		},
		"Job not marked for deletion": {
			job:    &batchv1.Job{},
			expect: "",
		},
		"Job exceeded deadline": {
			job: &batchv1.Job{
				Spec: batchv1.JobSpec{
					ActiveDeadlineSeconds: ptr.To(int64(123)),
				},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobFailed,
							Status: corev1.ConditionTrue,
							Reason: batchv1.JobReasonDeadlineExceeded,
						},
					},
				},
			},
			expect: "Job timed out after 123 seconds",
		},
		"Job terminated specifically": {
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						constants.AnnotationTerminationReason: "foobarbaz",
					},
				},
			},
			expect: "foobarbaz",
		},
		"Job terminated generically": {
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: ptr.To(metav1.NewTime(time.Now())),
				},
			},
			expect: controller.DefaultErrorMessage,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := watchers.GetJobError(test.job)
			if actual != test.expect {
				t.Errorf("Incorrect error message returned:\nexpect=%q\nactual=%q", test.expect, actual)
			}
		})
	}
}
