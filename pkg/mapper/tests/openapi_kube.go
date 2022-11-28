package tests

import (
	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MapToSpec maps TestUpsertRequest to Test CRD spec
func MapToSpec(request testkube.TestUpsertRequest) *testsv3.Test {

	test := &testsv3.Test{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsv3.TestSpec{
			Type_:            request.Type_,
			Content:          MapContentToSpecContent(request.Content),
			Source:           request.Source,
			Schedule:         request.Schedule,
			ExecutionRequest: MapExecutionRequestToSpecExecutionRequest(request.ExecutionRequest),
			Uploads:          request.Uploads,
		},
	}

	return test

}

// @Depracated
// MapDepratcatedParams maps old params to new variables data structure
func MapDepratcatedParams(in map[string]testkube.Variable) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v.Value
	}
	return out
}

// MapCRDVariables maps variables between API and operator CRDs
func MapCRDVariables(in map[string]testkube.Variable) map[string]testsv3.Variable {
	out := map[string]testsv3.Variable{}
	for k, v := range in {
		variable := testsv3.Variable{
			Name:  v.Name,
			Type_: string(*v.Type_),
			Value: v.Value,
		}

		if v.SecretRef != nil {
			variable.ValueFrom = corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: v.SecretRef.Name,
					},
					Key: v.SecretRef.Key,
				},
			}
		}

		out[k] = variable
	}
	return out
}

// MapContentToSpecContent maps TestContent OpenAPI spec to TestContent CRD spec
func MapContentToSpecContent(content *testkube.TestContent) (specContent *testsv3.TestContent) {
	if content == nil {
		return
	}

	var repository *testsv3.Repository
	if content.Repository != nil {
		repository = &testsv3.Repository{
			Type_:      content.Repository.Type_,
			Uri:        content.Repository.Uri,
			Branch:     content.Repository.Branch,
			Commit:     content.Repository.Commit,
			Path:       content.Repository.Path,
			WorkingDir: content.Repository.WorkingDir,
		}

		if content.Repository.UsernameSecret != nil {
			repository.UsernameSecret = &testsv3.SecretRef{
				Name: content.Repository.UsernameSecret.Name,
				Key:  content.Repository.UsernameSecret.Key,
			}
		}

		if content.Repository.TokenSecret != nil {
			repository.TokenSecret = &testsv3.SecretRef{
				Name: content.Repository.TokenSecret.Name,
				Key:  content.Repository.TokenSecret.Key,
			}
		}
	}

	return &testsv3.TestContent{
		Repository: repository,
		Data:       content.Data,
		Uri:        content.Uri,
		Type_:      content.Type_,
	}
}

// MapExecutionRequestToSpecExecutionRequest maps ExecutionRequest OpenAPI spec to ExecutionRequest CRD spec
func MapExecutionRequestToSpecExecutionRequest(executionRequest *testkube.ExecutionRequest) *testsv3.ExecutionRequest {
	if executionRequest == nil {
		return nil
	}

	var artifactRequest *testsv3.ArtifactRequest
	if executionRequest.ArtifactRequest != nil {
		artifactRequest = &testsv3.ArtifactRequest{
			StorageClassName: executionRequest.ArtifactRequest.StorageClassName,
			VolumeMountPath:  executionRequest.ArtifactRequest.VolumeMountPath,
			Dirs:             executionRequest.ArtifactRequest.Dirs,
		}
	}

	return &testsv3.ExecutionRequest{
		Name:                  executionRequest.Name,
		TestSuiteName:         executionRequest.TestSuiteName,
		Number:                int32(executionRequest.Number),
		ExecutionLabels:       executionRequest.ExecutionLabels,
		Namespace:             executionRequest.Namespace,
		VariablesFile:         executionRequest.VariablesFile,
		Variables:             MapCRDVariables(executionRequest.Variables),
		TestSecretUUID:        executionRequest.TestSecretUUID,
		TestSuiteSecretUUID:   executionRequest.TestSuiteSecretUUID,
		Args:                  executionRequest.Args,
		Envs:                  executionRequest.Envs,
		SecretEnvs:            executionRequest.SecretEnvs,
		Sync:                  executionRequest.Sync,
		HttpProxy:             executionRequest.HttpProxy,
		HttpsProxy:            executionRequest.HttpsProxy,
		Image:                 executionRequest.Image,
		ImagePullSecrets:      mapImagePullSecrets(executionRequest.ImagePullSecrets),
		ActiveDeadlineSeconds: executionRequest.ActiveDeadlineSeconds,
		Command:               executionRequest.Command,
		ArtifactRequest:       artifactRequest,
	}
}

func mapImagePullSecrets(secrets []testkube.LocalObjectReference) []v1.LocalObjectReference {
	var res []v1.LocalObjectReference
	for _, secret := range secrets {
		res = append(res, v1.LocalObjectReference{Name: secret.Name})
	}
	return res
}
