package watchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	constants2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestGetTerminationCode_DefaultAbortedWhenMissing(t *testing.T) {
	t.Parallel()

	assert.Equal(t, string(testkube.ABORTED_TestWorkflowStatus), GetTerminationCode(nil))
	assert.Equal(t, string(testkube.ABORTED_TestWorkflowStatus), GetTerminationCode(&batchv1.Job{}))
	assert.Equal(t, string(testkube.ABORTED_TestWorkflowStatus), GetTerminationCode(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}}))
}

func TestGetTerminationCode_FromAnnotation(t *testing.T) {
	t.Parallel()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				constants2.AnnotationTerminationCode: string(testkube.CANCELED_TestWorkflowStatus),
			},
		},
	}

	assert.Equal(t, string(testkube.CANCELED_TestWorkflowStatus), GetTerminationCode(job))
}
