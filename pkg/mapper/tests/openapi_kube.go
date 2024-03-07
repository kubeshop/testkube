package tests

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	mappertcl "github.com/kubeshop/testkube/pkg/tcl/mappertcl/tests"
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
			Description:      request.Description,
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
	if content.Repository != nil && !content.Repository.IsEmpty() {
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

		if content.Repository.UsernameSecret != nil && !content.Repository.UsernameSecret.IsEmpty() {
			repository.UsernameSecret = &testsv3.SecretRef{
				Name: content.Repository.UsernameSecret.Name,
				Key:  content.Repository.UsernameSecret.Key,
			}
		}

		if content.Repository.TokenSecret != nil && !content.Repository.TokenSecret.IsEmpty() {
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
			StorageClassName:       executionRequest.ArtifactRequest.StorageClassName,
			VolumeMountPath:        executionRequest.ArtifactRequest.VolumeMountPath,
			Dirs:                   executionRequest.ArtifactRequest.Dirs,
			Masks:                  executionRequest.ArtifactRequest.Masks,
			StorageBucket:          executionRequest.ArtifactRequest.StorageBucket,
			OmitFolderPerExecution: executionRequest.ArtifactRequest.OmitFolderPerExecution,
			SharedBetweenPods:      executionRequest.ArtifactRequest.SharedBetweenPods,
		}
	}

	var podRequest *testsv3.PodRequest
	if executionRequest.SlavePodRequest != nil {
		podRequest = &testsv3.PodRequest{}
		if executionRequest.SlavePodRequest.Resources != nil {
			podRequest.Resources = &testsv3.PodResourcesRequest{}
			if executionRequest.SlavePodRequest.Resources.Requests != nil {
				podRequest.Resources.Requests = &testsv3.ResourceRequest{
					Cpu:    executionRequest.SlavePodRequest.Resources.Requests.Cpu,
					Memory: executionRequest.SlavePodRequest.Resources.Requests.Memory,
				}
			}

			if executionRequest.SlavePodRequest.Resources.Limits != nil {
				podRequest.Resources.Limits = &testsv3.ResourceRequest{
					Cpu:    executionRequest.SlavePodRequest.Resources.Limits.Cpu,
					Memory: executionRequest.SlavePodRequest.Resources.Limits.Memory,
				}
			}
		}

		podRequest.PodTemplate = executionRequest.SlavePodRequest.PodTemplate
		podRequest.PodTemplateReference = executionRequest.SlavePodRequest.PodTemplateReference
	}

	result := &testsv3.ExecutionRequest{
		Name:                               executionRequest.Name,
		TestSuiteName:                      executionRequest.TestSuiteName,
		Number:                             executionRequest.Number,
		ExecutionLabels:                    executionRequest.ExecutionLabels,
		Namespace:                          executionRequest.Namespace,
		IsVariablesFileUploaded:            executionRequest.IsVariablesFileUploaded,
		VariablesFile:                      executionRequest.VariablesFile,
		Variables:                          MapCRDVariables(executionRequest.Variables),
		TestSecretUUID:                     executionRequest.TestSecretUUID,
		TestSuiteSecretUUID:                executionRequest.TestSuiteSecretUUID,
		Args:                               executionRequest.Args,
		ArgsMode:                           testsv3.ArgsModeType(executionRequest.ArgsMode),
		Envs:                               executionRequest.Envs,
		SecretEnvs:                         executionRequest.SecretEnvs,
		Sync:                               executionRequest.Sync,
		HttpProxy:                          executionRequest.HttpProxy,
		HttpsProxy:                         executionRequest.HttpsProxy,
		Image:                              executionRequest.Image,
		ImagePullSecrets:                   mapImagePullSecrets(executionRequest.ImagePullSecrets),
		ActiveDeadlineSeconds:              executionRequest.ActiveDeadlineSeconds,
		Command:                            executionRequest.Command,
		ArtifactRequest:                    artifactRequest,
		JobTemplate:                        executionRequest.JobTemplate,
		JobTemplateReference:               executionRequest.JobTemplateReference,
		CronJobTemplate:                    executionRequest.CronJobTemplate,
		CronJobTemplateReference:           executionRequest.CronJobTemplateReference,
		PreRunScript:                       executionRequest.PreRunScript,
		PostRunScript:                      executionRequest.PostRunScript,
		ExecutePostRunScriptBeforeScraping: executionRequest.ExecutePostRunScriptBeforeScraping,
		SourceScripts:                      executionRequest.SourceScripts,
		PvcTemplate:                        executionRequest.PvcTemplate,
		PvcTemplateReference:               executionRequest.PvcTemplateReference,
		ScraperTemplate:                    executionRequest.ScraperTemplate,
		ScraperTemplateReference:           executionRequest.ScraperTemplateReference,
		NegativeTest:                       executionRequest.NegativeTest,
		EnvConfigMaps:                      mapEnvReferences(executionRequest.EnvConfigMaps),
		EnvSecrets:                         mapEnvReferences(executionRequest.EnvSecrets),
		SlavePodRequest:                    podRequest,
	}

	// Pro edition only (tcl protected code)
	return mappertcl.MapExecutionRequestToSpecExecutionRequest(executionRequest, result)
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
			request.Description,
			&test.Spec.Description,
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
			executionRequest.JobTemplateReference,
			&request.JobTemplateReference,
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
			executionRequest.CronJobTemplateReference,
			&request.CronJobTemplateReference,
		},
		{
			executionRequest.PvcTemplate,
			&request.PvcTemplate,
		},
		{
			executionRequest.PvcTemplateReference,
			&request.PvcTemplateReference,
		},
		{
			executionRequest.ScraperTemplate,
			&request.ScraperTemplate,
		},
		{
			executionRequest.ScraperTemplateReference,
			&request.ScraperTemplateReference,
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

	if executionRequest.ExecutePostRunScriptBeforeScraping != nil {
		request.ExecutePostRunScriptBeforeScraping = *executionRequest.ExecutePostRunScriptBeforeScraping
		emptyExecution = false
	}

	// Pro edition only (tcl protected code)
	if !mappertcl.MapExecutionUpdateRequestToSpecExecutionRequest(executionRequest, request) {
		emptyExecution = false
	}

	if executionRequest.SourceScripts != nil {
		request.SourceScripts = *executionRequest.SourceScripts
	}

	if executionRequest.ArtifactRequest != nil {
		emptyArtifact := true
		if !(*executionRequest.ArtifactRequest == nil || (*executionRequest.ArtifactRequest).IsEmpty()) {
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

			if (*executionRequest.ArtifactRequest).Masks != nil {
				request.ArtifactRequest.Masks = *(*executionRequest.ArtifactRequest).Masks
				emptyArtifact = false
			}

			if (*executionRequest.ArtifactRequest).StorageBucket != nil {
				request.ArtifactRequest.StorageBucket = *(*executionRequest.ArtifactRequest).StorageBucket
				emptyArtifact = false
			}

			if (*executionRequest.ArtifactRequest).OmitFolderPerExecution != nil {
				request.ArtifactRequest.OmitFolderPerExecution = *(*executionRequest.ArtifactRequest).OmitFolderPerExecution
				emptyArtifact = false
			}

			if (*executionRequest.ArtifactRequest).SharedBetweenPods != nil {
				request.ArtifactRequest.SharedBetweenPods = *(*executionRequest.ArtifactRequest).SharedBetweenPods
				emptyArtifact = false
			}

		}

		if emptyArtifact {
			request.ArtifactRequest = nil
		} else {
			emptyExecution = false
		}
	}

	if executionRequest.SlavePodRequest != nil {
		emptyPodRequest := true
		if !(*executionRequest.SlavePodRequest == nil || (*executionRequest.SlavePodRequest).IsEmpty()) {
			if request.SlavePodRequest == nil {
				request.SlavePodRequest = &testsv3.PodRequest{}
			}

			if (*executionRequest.SlavePodRequest).Resources != nil {
				if request.SlavePodRequest.Resources == nil {
					request.SlavePodRequest.Resources = &testsv3.PodResourcesRequest{}
				}

				if (*(*executionRequest.SlavePodRequest).Resources).Requests != nil {
					if request.SlavePodRequest.Resources.Requests == nil {
						request.SlavePodRequest.Resources.Requests = &testsv3.ResourceRequest{}
					}

					if (*(*executionRequest.SlavePodRequest).Resources).Requests.Cpu != nil {
						request.SlavePodRequest.Resources.Requests.Cpu = *(*(*executionRequest.SlavePodRequest).Resources).Requests.Cpu
						emptyPodRequest = false
					}

					if (*(*executionRequest.SlavePodRequest).Resources).Requests.Memory != nil {
						request.SlavePodRequest.Resources.Requests.Memory = *(*(*executionRequest.SlavePodRequest).Resources).Requests.Memory
						emptyPodRequest = false
					}
				}

				if (*(*executionRequest.SlavePodRequest).Resources).Limits != nil {
					if request.SlavePodRequest.Resources.Limits == nil {
						request.SlavePodRequest.Resources.Limits = &testsv3.ResourceRequest{}
					}

					if (*(*executionRequest.SlavePodRequest).Resources).Limits.Cpu != nil {
						request.SlavePodRequest.Resources.Limits.Cpu = *(*(*executionRequest.SlavePodRequest).Resources).Limits.Cpu
						emptyPodRequest = false
					}

					if (*(*executionRequest.SlavePodRequest).Resources).Limits.Memory != nil {
						request.SlavePodRequest.Resources.Limits.Memory = *(*(*executionRequest.SlavePodRequest).Resources).Limits.Memory
						emptyPodRequest = false
					}
				}
			}

			if (*executionRequest.SlavePodRequest).PodTemplate != nil {
				request.SlavePodRequest.PodTemplate = *(*executionRequest.SlavePodRequest).PodTemplate
				emptyPodRequest = false
			}

			if (*executionRequest.SlavePodRequest).PodTemplateReference != nil {
				request.SlavePodRequest.PodTemplateReference = *(*executionRequest.SlavePodRequest).PodTemplateReference
				emptyPodRequest = false
			}
		}

		if emptyPodRequest {
			request.SlavePodRequest = nil
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
