package executors

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapCRDToAPI maps Executor CRD to OpenAPI spec Executor
func MapCRDToAPI(item executorv1.Executor) testkube.ExecutorUpsertRequest {
	return testkube.ExecutorUpsertRequest{
		Name:             item.Name,
		Namespace:        item.Namespace,
		Labels:           item.Labels,
		ExecutorType:     item.Spec.ExecutorType,
		Types:            item.Spec.Types,
		Uri:              item.Spec.URI,
		Image:            item.Spec.Image,
		ImagePullSecrets: mapImagePullSecretsToAPI(item.Spec.ImagePullSecrets),
		Command:          item.Spec.Command,
		Args:             item.Spec.Args,
		JobTemplate:      item.Spec.JobTemplate,
		Features:         mapFeaturesToAPI(item.Spec.Features),
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
			ImagePullSecrets: mapImagePullSecretsToCRD(request.ImagePullSecrets),
			Command:          request.Command,
			Args:             request.Args,
			JobTemplate:      request.JobTemplate,
			Features:         mapFeaturesToCRD(request.Features),
		},
	}
}

// MapExecutorCRDToExecutorDetails maps CRD Executor to OpemAPI spec ExecutorDetails
func MapExecutorCRDToExecutorDetails(item executorv1.Executor) testkube.ExecutorDetails {
	return testkube.ExecutorDetails{
		Name: item.Name,
		Executor: &testkube.Executor{
			ExecutorType:     item.Spec.ExecutorType,
			Image:            item.Spec.Image,
			ImagePullSecrets: mapImagePullSecretsToAPI(item.Spec.ImagePullSecrets),
			Command:          item.Spec.Command,
			Args:             item.Spec.Args,
			Types:            item.Spec.Types,
			Uri:              item.Spec.URI,
			JobTemplate:      item.Spec.JobTemplate,
			Labels:           item.Labels,
			Features:         mapFeaturesToAPI(item.Spec.Features),
		},
	}
}

func mapImagePullSecretsToCRD(secrets []testkube.LocalObjectReference) []v1.LocalObjectReference {
	var res []v1.LocalObjectReference
	for _, secret := range secrets {
		res = append(res, v1.LocalObjectReference{Name: secret.Name})
	}
	return res
}

func mapImagePullSecretsToAPI(secrets []v1.LocalObjectReference) []testkube.LocalObjectReference {
	var res []testkube.LocalObjectReference
	for _, secret := range secrets {
		res = append(res, testkube.LocalObjectReference{Name: secret.Name})
	}
	return res
}

func mapFeaturesToCRD(features []string) (out []executorv1.Feature) {
	for _, feature := range features {
		out = append(out, executorv1.Feature(feature))
	}
	return out
}

func mapFeaturesToAPI(features []executorv1.Feature) (out []string) {
	for _, feature := range features {
		out = append(out, string(feature))
	}
	return out
}

// MapUpdateToSpec maps ExecutorUpdateRequest to Executor CRD spec
func MapUpdateToSpec(request testkube.ExecutorUpdateRequest, executor *executorv1.Executor) *executorv1.Executor {
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			request.Name,
			&executor.Name,
		},
		{
			request.Namespace,
			&executor.Namespace,
		},
		{
			request.ExecutorType,
			&executor.Spec.ExecutorType,
		},
		{
			request.Image,
			&executor.Spec.Image,
		},
		{
			request.Uri,
			&executor.Spec.URI,
		},
		{
			request.JobTemplate,
			&executor.Spec.JobTemplate,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
		}
	}

	var slices = []struct {
		source      *[]string
		destination *[]string
	}{
		{
			request.Command,
			&executor.Spec.Command,
		},
		{
			request.Args,
			&executor.Spec.Args,
		},
		{
			request.Types,
			&executor.Spec.Types,
		},
	}

	for _, slice := range slices {
		if slice.source != nil {
			*slice.destination = *slice.source
		}
	}

	if request.Labels != nil {
		executor.Labels = *request.Labels
	}

	if request.ImagePullSecrets != nil {
		executor.Spec.ImagePullSecrets = mapImagePullSecretsToCRD(*request.ImagePullSecrets)
	}

	if request.Features != nil {
		executor.Spec.Features = mapFeaturesToCRD(*request.Features)
	}

	return executor
}
