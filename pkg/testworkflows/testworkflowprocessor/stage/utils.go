package stage

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	quantity "k8s.io/apimachinery/pkg/api/resource"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

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
