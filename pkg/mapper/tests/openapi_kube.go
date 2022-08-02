package tests

import (
	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
			Schedule:         request.Schedule,
			ExecutionRequest: MapExecutionRequestToSpecExecutionRequest(request.ExecutionRequest),
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
		out[k] = testsv3.Variable{
			Name:  v.Name,
			Type_: string(*v.Type_),
			Value: v.Value,
		}
	}
	return out
}

// MapContentToSpecContent maps TestContent OpenAPI spec to TestContent CRD spec
func MapContentToSpecContent(content *testkube.TestContent) (specContent *testsv3.TestContent) {
	if content == nil {
		return
	}

	return &testsv3.TestContent{
		// assuming same data structure
		Repository: (*testsv3.Repository)(content.Repository),
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

	return &testsv3.ExecutionRequest{
		Name:                executionRequest.Name,
		TestSuiteName:       executionRequest.TestSuiteName,
		Number:              executionRequest.Number,
		ExecutionLabels:     executionRequest.ExecutionLabels,
		Namespace:           executionRequest.Namespace,
		VariablesFile:       executionRequest.VariablesFile,
		Variables:           MapCRDVariables(executionRequest.Variables),
		TestSecretUUID:      executionRequest.TestSecretUUID,
		TestSuiteSecretUUID: executionRequest.TestSuiteSecretUUID,
		Args:                executionRequest.Args,
		Envs:                executionRequest.Envs,
		SecretEnvs:          executionRequest.SecretEnvs,
		Sync:                executionRequest.Sync,
		HttpProxy:           executionRequest.HttpProxy,
		HttpsProxy:          executionRequest.HttpsProxy,
	}
}
