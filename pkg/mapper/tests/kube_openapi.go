package tests

import (
	commonv1 "github.com/kubeshop/testkube-operator/apis/common/v1"
	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapTestListKubeToAPI maps CRD list data to OpenAPI spec tests list
func MapTestListKubeToAPI(crTests testsv3.TestList) (tests []testkube.Test) {
	tests = []testkube.Test{}
	for _, item := range crTests.Items {
		tests = append(tests, MapTestCRToAPI(item))
	}

	return
}

// MapTestCRToAPI maps CRD to OpenAPI spec test
func MapTestCRToAPI(crTest testsv3.Test) (test testkube.Test) {
	test.Name = crTest.Name
	test.Namespace = crTest.Namespace
	test.Content = MapTestContentFromSpec(crTest.Spec.Content)
	test.Created = crTest.CreationTimestamp.Time
	test.Type_ = crTest.Spec.Type_
	test.Labels = crTest.Labels
	test.Schedule = crTest.Spec.Schedule
	test.ExecutionRequest = MapExecutionRequestFromSpec(crTest.Spec.ExecutionRequest)
	return
}

func MergeVariablesAndParams(variables map[string]testsv3.Variable, params map[string]string) map[string]testkube.Variable {
	out := map[string]testkube.Variable{}
	for k, v := range params {
		out[k] = testkube.NewBasicVariable(k, v)
	}

	for k, v := range variables {
		if v.Type_ == commonv1.VariableTypeSecret {
			out[k] = testkube.NewSecretVariable(v.Name, v.Value)
		}
		if v.Type_ == commonv1.VariableTypeBasic {
			out[k] = testkube.NewBasicVariable(v.Name, v.Value)
		}
	}

	return out
}

// MapTestContentFromSpec maps CRD to OpenAPI spec TestContent
func MapTestContentFromSpec(specContent *testsv3.TestContent) *testkube.TestContent {

	content := &testkube.TestContent{}
	if specContent != nil {
		content.Type_ = specContent.Type_
		// assuming same data structure - there is task about syncing them automatically
		// https://github.com/kubeshop/testkube/issues/723
		content.Repository = (*testkube.Repository)(specContent.Repository)
		content.Data = specContent.Data
		content.Uri = specContent.Uri
	}

	return content
}

// MapTestArrayKubeToAPI maps CRD array data to OpenAPI spec tests list
func MapTestArrayKubeToAPI(crTests []testsv3.Test) (tests []testkube.Test) {
	tests = []testkube.Test{}
	for _, crTest := range crTests {
		tests = append(tests, MapTestCRToAPI(crTest))
	}

	return
}

// MapExecutionRequestFromSpec maps CRD to OpenAPI spec ExecutionREquest
func MapExecutionRequestFromSpec(specExecutionRequest *testsv3.ExecutionRequest) *testkube.ExecutionRequest {
	if specExecutionRequest == nil {
		return nil
	}

	return &testkube.ExecutionRequest{
		Name:                specExecutionRequest.Name,
		TestSuiteName:       specExecutionRequest.TestSuiteName,
		Number:              specExecutionRequest.Number,
		ExecutionLabels:     specExecutionRequest.ExecutionLabels,
		Namespace:           specExecutionRequest.Namespace,
		VariablesFile:       specExecutionRequest.VariablesFile,
		Variables:           MergeVariablesAndParams(specExecutionRequest.Variables, nil),
		TestSecretUUID:      specExecutionRequest.TestSecretUUID,
		TestSuiteSecretUUID: specExecutionRequest.TestSuiteSecretUUID,
		Args:                specExecutionRequest.Args,
		Envs:                specExecutionRequest.Envs,
		SecretEnvs:          specExecutionRequest.SecretEnvs,
		Sync:                specExecutionRequest.Sync,
		HttpProxy:           specExecutionRequest.HttpProxy,
		HttpsProxy:          specExecutionRequest.HttpsProxy,
	}
}
