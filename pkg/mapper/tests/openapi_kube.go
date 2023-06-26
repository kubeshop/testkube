package tests

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapUpsertToSpec maps TestUpsertRequest to Test CRD spec
func MapUpsertToSpec(request testkube.TestUpsertRequest) *testsv3.Test {

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

		if v.ConfigMapRef != nil {
			variable.ValueFrom = corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: v.ConfigMapRef.Name,
					},
					Key: v.ConfigMapRef.Key,
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
			Type_:             content.Repository.Type_,
			Uri:               content.Repository.Uri,
			Branch:            content.Repository.Branch,
			Commit:            content.Repository.Commit,
			Path:              content.Repository.Path,
			WorkingDir:        content.Repository.WorkingDir,
			CertificateSecret: content.Repository.CertificateSecret,
			AuthType:          testsv3.GitAuthType(content.Repository.AuthType),
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
		Type_:      testsv3.TestContentType(content.Type_),
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
		Name:                    executionRequest.Name,
		TestSuiteName:           executionRequest.TestSuiteName,
		Number:                  executionRequest.Number,
		ExecutionLabels:         executionRequest.ExecutionLabels,
		Namespace:               executionRequest.Namespace,
		IsVariablesFileUploaded: executionRequest.IsVariablesFileUploaded,
		VariablesFile:           executionRequest.VariablesFile,
		Variables:               MapCRDVariables(executionRequest.Variables),
		TestSecretUUID:          executionRequest.TestSecretUUID,
		TestSuiteSecretUUID:     executionRequest.TestSuiteSecretUUID,
		Args:                    executionRequest.Args,
		ArgsMode:                testsv3.ArgsModeType(executionRequest.ArgsMode),
		Envs:                    executionRequest.Envs,
		SecretEnvs:              executionRequest.SecretEnvs,
		Sync:                    executionRequest.Sync,
		HttpProxy:               executionRequest.HttpProxy,
		HttpsProxy:              executionRequest.HttpsProxy,
		Image:                   executionRequest.Image,
		ImagePullSecrets:        mapImagePullSecrets(executionRequest.ImagePullSecrets),
		ActiveDeadlineSeconds:   executionRequest.ActiveDeadlineSeconds,
		Command:                 executionRequest.Command,
		ArtifactRequest:         artifactRequest,
		JobTemplate:             executionRequest.JobTemplate,
		CronJobTemplate:         executionRequest.CronJobTemplate,
		PreRunScript:            executionRequest.PreRunScript,
		PostRunScript:           executionRequest.PostRunScript,
		ScraperTemplate:         executionRequest.ScraperTemplate,
		NegativeTest:            executionRequest.NegativeTest,
		EnvConfigMaps:           mapEnvReferences(executionRequest.EnvConfigMaps),
		EnvSecrets:              mapEnvReferences(executionRequest.EnvSecrets),
	}
}

func mapImagePullSecrets(secrets []testkube.LocalObjectReference) (res []v1.LocalObjectReference) {
	for _, secret := range secrets {
		res = append(res, v1.LocalObjectReference{Name: secret.Name})
	}
	return res
}

func mapEnvReferences(envs []testkube.EnvReference) []testsv3.EnvReference {
	if envs == nil {
		return nil
	}
	var res []testsv3.EnvReference
	for _, env := range envs {
		if env.Reference == nil {
			continue
		}

		res = append(res, testsv3.EnvReference{
			LocalObjectReference: v1.LocalObjectReference{
				Name: env.Reference.Name,
			},
			Mount:          env.Mount,
			MountPath:      env.MountPath,
			MapToVariables: env.MapToVariables,
		})
	}

	return res
}

// MapUpdateToSpec maps TestUpdateRequest to Test CRD spec
func MapUpdateToSpec(request testkube.TestUpdateRequest, test *testsv3.Test) *testsv3.Test {
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			request.Name,
			&test.Name,
		},
		{
			request.Namespace,
			&test.Namespace,
		},
		{
			request.Type_,
			&test.Spec.Type_,
		},
		{
			request.Source,
			&test.Spec.Source,
		},
		{
			request.Schedule,
			&test.Spec.Schedule,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
		}
	}

	if request.Content != nil {
		test.Spec.Content = MapUpdateContentToSpecContent(*request.Content, test.Spec.Content)
	}

	if request.ExecutionRequest != nil {
		test.Spec.ExecutionRequest = MapExecutionUpdateRequestToSpecExecutionRequest(*request.ExecutionRequest, test.Spec.ExecutionRequest)
	}

	if request.Labels != nil {
		test.Labels = *request.Labels
	}

	if request.Uploads != nil {
		test.Spec.Uploads = *request.Uploads
	}

	return test
}

// MapUpdateContentToSpecContent maps TestUpdateContent OpenAPI spec to TestContent CRD spec
func MapUpdateContentToSpecContent(content *testkube.TestContentUpdate, testContent *testsv3.TestContent) *testsv3.TestContent {
	if content == nil {
		return nil
	}

	if testContent == nil {
		testContent = &testsv3.TestContent{}
	}

	emptyContent := true
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			content.Data,
			&testContent.Data,
		},
		{
			content.Uri,
			&testContent.Uri,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
			emptyContent = false
		}
	}

	if content.Type_ != nil {
		testContent.Type_ = testsv3.TestContentType(*content.Type_)
		emptyContent = false
	}

	if content.Repository != nil {
		if *content.Repository == nil {
			testContent.Repository = nil
			return testContent
		}

		if (*content.Repository).IsEmpty() {
			testContent.Repository = nil
			return testContent
		}

		if testContent.Repository == nil {
			testContent.Repository = &testsv3.Repository{}
		}

		emptyRepository := true
		fake := ""
		var fields = []struct {
			source      *string
			destination *string
		}{
			{
				(*content.Repository).Type_,
				&testContent.Repository.Type_,
			},
			{
				(*content.Repository).Uri,
				&testContent.Repository.Uri,
			},
			{
				(*content.Repository).Branch,
				&testContent.Repository.Branch,
			},
			{
				(*content.Repository).Commit,
				&testContent.Repository.Commit,
			},
			{
				(*content.Repository).Path,
				&testContent.Repository.Path,
			},
			{
				(*content.Repository).WorkingDir,
				&testContent.Repository.WorkingDir,
			},
			{
				(*content.Repository).CertificateSecret,
				&testContent.Repository.CertificateSecret,
			},
			{
				(*content.Repository).Username,
				&fake,
			},
			{
				(*content.Repository).Token,
				&fake,
			},
		}

		for _, field := range fields {
			if field.source != nil {
				*field.destination = *field.source
				emptyRepository = false
			}
		}

		if (*content.Repository).AuthType != nil {
			testContent.Repository.AuthType = testsv3.GitAuthType(*(*content.Repository).AuthType)
			emptyRepository = false
		}

		if (*content.Repository).UsernameSecret != nil {
			if (*(*content.Repository).UsernameSecret).IsEmpty() {
				testContent.Repository.UsernameSecret = nil
			} else {
				testContent.Repository.UsernameSecret = &testsv3.SecretRef{
					Name: (*(*content.Repository).UsernameSecret).Name,
					Key:  (*(*content.Repository).UsernameSecret).Key,
				}
			}

			emptyRepository = false
		}

		if (*content.Repository).TokenSecret != nil {
			if (*(*content.Repository).TokenSecret).IsEmpty() {
				testContent.Repository.TokenSecret = nil
			} else {
				testContent.Repository.TokenSecret = &testsv3.SecretRef{
					Name: (*(*content.Repository).TokenSecret).Name,
					Key:  (*(*content.Repository).TokenSecret).Key,
				}
			}

			emptyRepository = false
		}

		if emptyRepository {
			testContent.Repository = nil
		} else {
			emptyContent = false
		}
	}

	if emptyContent {
		return nil
	}

	return testContent
}

// MapExecutionUpdateRequestToSpecExecutionRequest maps ExecutionUpdateRequest OpenAPI spec to ExecutionRequest CRD spec
func MapExecutionUpdateRequestToSpecExecutionRequest(executionRequest *testkube.ExecutionUpdateRequest,
	request *testsv3.ExecutionRequest) *testsv3.ExecutionRequest {
	if executionRequest == nil {
		return nil
	}

	if request == nil {
		request = &testsv3.ExecutionRequest{}
	}

	emptyExecution := true
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			executionRequest.Name,
			&request.Name,
		},
		{
			executionRequest.TestSuiteName,
			&request.TestSuiteName,
		},
		{
			executionRequest.Namespace,
			&request.Namespace,
		},
		{
			executionRequest.VariablesFile,
			&request.VariablesFile,
		},
		{
			executionRequest.TestSecretUUID,
			&request.TestSecretUUID,
		},
		{
			executionRequest.TestSuiteSecretUUID,
			&request.TestSuiteSecretUUID,
		},
		{
			executionRequest.HttpProxy,
			&request.HttpProxy,
		},
		{
			executionRequest.HttpsProxy,
			&request.HttpsProxy,
		},
		{
			executionRequest.Image,
			&request.Image,
		},
		{
			executionRequest.JobTemplate,
			&request.JobTemplate,
		},
		{
			executionRequest.PreRunScript,
			&request.PreRunScript,
		},
		{
			executionRequest.PostRunScript,
			&request.PostRunScript,
		},
		{
			executionRequest.CronJobTemplate,
			&request.CronJobTemplate,
		},
		{
			executionRequest.ScraperTemplate,
			&request.ScraperTemplate,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
			emptyExecution = false
		}
	}

	if executionRequest.ArgsMode != nil {
		request.ArgsMode = testsv3.ArgsModeType(*executionRequest.ArgsMode)
		emptyExecution = false
	}

	var slices = []struct {
		source      *map[string]string
		destination *map[string]string
	}{
		{
			executionRequest.ExecutionLabels,
			&request.ExecutionLabels,
		},
		{
			executionRequest.Envs,
			&request.Envs,
		},
		{
			executionRequest.SecretEnvs,
			&request.SecretEnvs,
		},
	}

	for _, slice := range slices {
		if slice.source != nil {
			*slice.destination = *slice.source
			emptyExecution = false
		}
	}

	if executionRequest.Number != nil {
		request.Number = *executionRequest.Number
		emptyExecution = false
	}

	if executionRequest.Sync != nil {
		request.Sync = *executionRequest.Sync
		emptyExecution = false
	}

	if executionRequest.NegativeTest != nil {
		request.NegativeTest = *executionRequest.NegativeTest
		emptyExecution = false
	}

	if executionRequest.ActiveDeadlineSeconds != nil {
		request.ActiveDeadlineSeconds = *executionRequest.ActiveDeadlineSeconds
		emptyExecution = false
	}

	if executionRequest.Args != nil {
		request.Args = *executionRequest.Args
		emptyExecution = false
	}

	if executionRequest.Command != nil {
		request.Command = *executionRequest.Command
		emptyExecution = false
	}

	if executionRequest.Variables != nil {
		request.Variables = MapCRDVariables(*executionRequest.Variables)
		emptyExecution = false
	}

	if executionRequest.ImagePullSecrets != nil {
		request.ImagePullSecrets = mapImagePullSecrets(*executionRequest.ImagePullSecrets)
		emptyExecution = false
	}

	if executionRequest.EnvConfigMaps != nil {
		request.EnvConfigMaps = mapEnvReferences(*executionRequest.EnvConfigMaps)
		emptyExecution = false
	}

	if executionRequest.EnvSecrets != nil {
		request.EnvSecrets = mapEnvReferences(*executionRequest.EnvSecrets)
		emptyExecution = false
	}

	if executionRequest.ArtifactRequest != nil {
		if *executionRequest.ArtifactRequest == nil {
			request.ArtifactRequest = nil
			return request
		}

		if (*executionRequest.ArtifactRequest).IsEmpty() {
			request.ArtifactRequest = nil
			return request
		}

		emptyArtifact := true
		if request.ArtifactRequest == nil {
			request.ArtifactRequest = &testsv3.ArtifactRequest{}
		}

		if (*executionRequest.ArtifactRequest).StorageClassName != nil {
			request.ArtifactRequest.StorageClassName = *(*executionRequest.ArtifactRequest).StorageClassName
			emptyArtifact = false
		}

		if (*executionRequest.ArtifactRequest).VolumeMountPath != nil {
			request.ArtifactRequest.VolumeMountPath = *(*executionRequest.ArtifactRequest).VolumeMountPath
			emptyArtifact = false
		}

		if (*executionRequest.ArtifactRequest).Dirs != nil {
			request.ArtifactRequest.Dirs = *(*executionRequest.ArtifactRequest).Dirs
			emptyArtifact = false
		}

		if emptyArtifact {
			request.ArtifactRequest = nil
		} else {
			emptyExecution = false
		}
	}

	if emptyExecution {
		return nil
	}

	return request
}

// MapStatusToSpec maps OpenAPI spec TestStatus to CRD
func MapStatusToSpec(testStatus *testkube.TestStatus) (specStatus testsv3.TestStatus) {
	if testStatus == nil || testStatus.LatestExecution == nil {
		return specStatus
	}

	specStatus.LatestExecution = &testsv3.ExecutionCore{
		Id:     testStatus.LatestExecution.Id,
		Number: testStatus.LatestExecution.Number,
		Status: (*testsv3.ExecutionStatus)(testStatus.LatestExecution.Status),
	}

	specStatus.LatestExecution.StartTime.Time = testStatus.LatestExecution.StartTime
	specStatus.LatestExecution.EndTime.Time = testStatus.LatestExecution.EndTime

	return specStatus
}

// MapExecutionToTestStatus maps OpenAPI Execution to TestStatus CRD
func MapExecutionToTestStatus(execution *testkube.Execution) (specStatus testsv3.TestStatus) {
	specStatus.LatestExecution = &testsv3.ExecutionCore{
		Id:     execution.Id,
		Number: execution.Number,
	}

	if execution.ExecutionResult != nil {
		specStatus.LatestExecution.Status = (*testsv3.ExecutionStatus)(execution.ExecutionResult.Status)
	}

	specStatus.LatestExecution.StartTime.Time = execution.StartTime
	specStatus.LatestExecution.EndTime.Time = execution.EndTime

	return specStatus
}

// MapTestSuiteExecutionStatusToExecutionStatus maps test suite execution status to execution status
func MapTestSuiteExecutionStatusToExecutionStatus(testSuiteStatus *testkube.TestSuiteExecutionStatus) (
	testStatus *testkube.ExecutionStatus) {
	switch testSuiteStatus {
	case testkube.TestSuiteExecutionStatusAborted:
		testStatus = testkube.ExecutionStatusAborted
	case testkube.TestSuiteExecutionStatusTimeout:
		testStatus = testkube.ExecutionStatusTimeout
	case testkube.TestSuiteExecutionStatusRunning:
		testStatus = testkube.ExecutionStatusRunning
	case testkube.TestSuiteExecutionStatusQueued:
		testStatus = testkube.ExecutionStatusQueued
	case testkube.TestSuiteExecutionStatusFailed:
		testStatus = testkube.ExecutionStatusFailed
	case testkube.TestSuiteExecutionStatusPassed:
		testStatus = testkube.ExecutionStatusPassed

	}

	return testStatus
}
