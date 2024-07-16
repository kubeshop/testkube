package testworkflowprocessor

import (
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

var BypassToolkitCheck = corev1.EnvVar{
	Name:  "TK_TC_SECURITY",
	Value: rand.String(20),
}

func MapResourcesToKubernetesResources(resources *testworkflowsv1.Resources) (corev1.ResourceRequirements, error) {
	result := corev1.ResourceRequirements{}
	if resources != nil {
		if len(resources.Requests) > 0 {
			result.Requests = make(corev1.ResourceList)
		}
		if len(resources.Limits) > 0 {
			result.Limits = make(corev1.ResourceList)
		}
		for k, v := range resources.Requests {
			var err error
			result.Requests[k], err = quantity.ParseQuantity(v.String())
			if err != nil {
				return corev1.ResourceRequirements{}, errors.Wrap(err, "parsing resources")
			}
		}
		for k, v := range resources.Limits {
			var err error
			result.Limits[k], err = quantity.ParseQuantity(v.String())
			if err != nil {
				return corev1.ResourceRequirements{}, errors.Wrap(err, "parsing resources")
			}
		}
	}
	return result, nil
}

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
