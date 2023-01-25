package tests

import (
	v1 "k8s.io/api/core/v1"

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
	test.Source = crTest.Spec.Source
	test.Type_ = crTest.Spec.Type_
	test.Labels = crTest.Labels
	test.Schedule = crTest.Spec.Schedule
	test.ExecutionRequest = MapExecutionRequestFromSpec(crTest.Spec.ExecutionRequest)
	test.Uploads = crTest.Spec.Uploads
	test.Status = MapStatusFromSpec(crTest.Status)
	return
}

func MergeVariablesAndParams(variables map[string]testsv3.Variable, params map[string]string) map[string]testkube.Variable {
	out := map[string]testkube.Variable{}
	for k, v := range params {
		out[k] = testkube.NewBasicVariable(k, v)
	}

	for k, v := range variables {
		if v.Type_ == commonv1.VariableTypeSecret {
			if v.ValueFrom.SecretKeyRef == nil {
				out[k] = testkube.NewSecretVariable(v.Name, v.Value)
			} else {
				out[k] = testkube.NewSecretVariableReference(v.Name, v.ValueFrom.SecretKeyRef.Name, v.ValueFrom.SecretKeyRef.Key)
			}
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
		content.Data = specContent.Data
		content.Uri = specContent.Uri
		if specContent.Repository != nil {
			content.Repository = &testkube.Repository{
				Type_:             specContent.Repository.Type_,
				Uri:               specContent.Repository.Uri,
				Branch:            specContent.Repository.Branch,
				Commit:            specContent.Repository.Commit,
				Path:              specContent.Repository.Path,
				WorkingDir:        specContent.Repository.WorkingDir,
				CertificateSecret: specContent.Repository.CertificateSecret,
			}

			if specContent.Repository.UsernameSecret != nil {
				content.Repository.UsernameSecret = &testkube.SecretRef{
					Name: specContent.Repository.UsernameSecret.Name,
					Key:  specContent.Repository.UsernameSecret.Key,
				}
			}

			if specContent.Repository.TokenSecret != nil {
				content.Repository.TokenSecret = &testkube.SecretRef{
					Name: specContent.Repository.TokenSecret.Name,
					Key:  specContent.Repository.TokenSecret.Key,
				}
			}
		}
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

	var artifactRequest *testkube.ArtifactRequest
	if specExecutionRequest.ArtifactRequest != nil {
		artifactRequest = &testkube.ArtifactRequest{
			StorageClassName: specExecutionRequest.ArtifactRequest.StorageClassName,
			VolumeMountPath:  specExecutionRequest.ArtifactRequest.VolumeMountPath,
			Dirs:             specExecutionRequest.ArtifactRequest.Dirs,
		}
	}

	return &testkube.ExecutionRequest{
		Name:                  specExecutionRequest.Name,
		TestSuiteName:         specExecutionRequest.TestSuiteName,
		Number:                specExecutionRequest.Number,
		ExecutionLabels:       specExecutionRequest.ExecutionLabels,
		Namespace:             specExecutionRequest.Namespace,
		VariablesFile:         specExecutionRequest.VariablesFile,
		Variables:             MergeVariablesAndParams(specExecutionRequest.Variables, nil),
		TestSecretUUID:        specExecutionRequest.TestSecretUUID,
		TestSuiteSecretUUID:   specExecutionRequest.TestSuiteSecretUUID,
		Command:               specExecutionRequest.Command,
		Args:                  specExecutionRequest.Args,
		Image:                 specExecutionRequest.Image,
		ImagePullSecrets:      MapImagePullSecrets(specExecutionRequest.ImagePullSecrets),
		Envs:                  specExecutionRequest.Envs,
		SecretEnvs:            specExecutionRequest.SecretEnvs,
		Sync:                  specExecutionRequest.Sync,
		HttpProxy:             specExecutionRequest.HttpProxy,
		HttpsProxy:            specExecutionRequest.HttpsProxy,
		ActiveDeadlineSeconds: specExecutionRequest.ActiveDeadlineSeconds,
		ArtifactRequest:       artifactRequest,
		JobTemplate:           specExecutionRequest.JobTemplate,
		PreRunScript:          specExecutionRequest.PreRunScript,
		ScraperTemplate:       specExecutionRequest.ScraperTemplate,
		NegativeTest:          specExecutionRequest.NegativeTest,
	}
}

// MapImagePullSecrets maps Kubernetes spec to testkube model
func MapImagePullSecrets(lor []v1.LocalObjectReference) []testkube.LocalObjectReference {
	if lor == nil {
		return nil
	}
	var res []testkube.LocalObjectReference
	for _, ref := range lor {
		res = append(res, testkube.LocalObjectReference{Name: ref.Name})
	}

	return res
}

// MapStatusFromSpec maps CRD to OpenAPI spec TestStatus
func MapStatusFromSpec(specStatus testsv3.TestStatus) *testkube.TestStatus {
	if specStatus.LatestExecution == nil {
		return nil
	}

	return &testkube.TestStatus{
		LatestExecution: &testkube.ExecutionCore{
			Id:        specStatus.LatestExecution.Id,
			Number:    specStatus.LatestExecution.Number,
			Status:    (*testkube.ExecutionStatus)(specStatus.LatestExecution.Status),
			StartTime: specStatus.LatestExecution.StartTime.Time,
			EndTime:   specStatus.LatestExecution.EndTime.Time,
		},
	}
}
