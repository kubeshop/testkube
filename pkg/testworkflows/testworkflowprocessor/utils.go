package testworkflowprocessor

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func AnnotateControlledBy(obj metav1.Object, rootId, id string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constants.RootResourceIdLabelName] = rootId
	labels[constants.ResourceIdLabelName] = id
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateControlledBy(&v.Spec.Template, rootId, id)
	}
}

func AnnotateGroupId(obj metav1.Object, id string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constants.GroupIdLabelName] = id
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateGroupId(&v.Spec.Template, id)
	}
}

func AnnotateRunnerId(obj metav1.Object, id string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constants.RunnerIdLabelName] = id
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateRunnerId(&v.Spec.Template, id)
	}
}
