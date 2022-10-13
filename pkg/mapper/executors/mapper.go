package executors

import (
	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MapCRDToAPI maps Executor CRD to OpenAPI spec Webhook
func MapCRDToAPI(item executorv1.Executor) testkube.ExecutorUpsertRequest {
	return testkube.ExecutorUpsertRequest{
		Name:         item.Name,
		Namespace:    item.Namespace,
		Labels:       item.Labels,
		ExecutorType: item.Spec.ExecutorType,
		Types:        item.Spec.Types,
		Uri:          item.Spec.URI,
		Image:        item.Spec.Image,
		JobTemplate:  item.Spec.JobTemplate,
	}
}

// MapAPIToCRD maps OpenAPI spec ExecutorUpsertRequest to CRD Executor
func MapAPIToCRD(request testkube.ExecutorUpsertRequest) executorv1.Executor {
	return executorv1.Executor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: executorv1.ExecutorSpec{
			ExecutorType:     request.ExecutorType,
			Types:            request.Types,
			URI:              request.Uri,
			Image:            request.Image,
			ImagePullSecrets: mapImagePullSecrets(request.ImagePullSecrets),
			Command:          request.Command,
			Args:             request.Args,
			JobTemplate:      request.JobTemplate,
		},
	}
}

func mapImagePullSecrets(secrets []testkube.LocalObjectReference) []v1.LocalObjectReference {
	var res []v1.LocalObjectReference
	for _, secret := range secrets {
		res = append(res, v1.LocalObjectReference{Name: secret.Name})
	}
	return res
}
