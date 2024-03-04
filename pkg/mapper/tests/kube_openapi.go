package tests

import (
	v1 "k8s.io/api/core/v1"

	commonv1 "github.com/kubeshop/testkube-operator/api/common/v1"
	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	mappertcl "github.com/kubeshop/testkube/pkg/tcl/mappertcl/tests"
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
	test.Description = crTest.Spec.Description
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
			if v.ValueFrom.ConfigMapKeyRef == nil {
				out[k] = testkube.NewBasicVariable(v.Name, v.Value)
			} else {
				out[k] = testkube.NewConfigMapVariableReference(v.Name, v.ValueFrom.ConfigMapKeyRef.Name, v.ValueFrom.ConfigMapKeyRef.Key)
			}
		}
	}

	return out
}

// MapTestContentFromSpec maps CRD to OpenAPI spec TestContent
func MapTestContentFromSpec(specContent *testsv3.TestContent) *testkube.TestContent {

	content := &testkube.TestContent{}
	if specContent != nil {
		content.Type_ = string(specContent.Type_)
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
				AuthType:          string(specContent.Repository.AuthType),
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
			StorageClassName:       specExecutionRequest.ArtifactRequest.StorageClassName,
			VolumeMountPath:        specExecutionRequest.ArtifactRequest.VolumeMountPath,
			Dirs:                   specExecutionRequest.ArtifactRequest.Dirs,
			Masks:                  specExecutionRequest.ArtifactRequest.Masks,
			StorageBucket:          specExecutionRequest.ArtifactRequest.StorageBucket,
			OmitFolderPerExecution: specExecutionRequest.ArtifactRequest.OmitFolderPerExecution,
			SharedBetweenPods:      specExecutionRequest.ArtifactRequest.SharedBetweenPods,
		}
	}

	var podRequest *testkube.PodRequest
	if specExecutionRequest.SlavePodRequest != nil {
		podRequest = &testkube.PodRequest{}
		if specExecutionRequest.SlavePodRequest.Resources != nil {
			podRequest.Resources = &testkube.PodResourcesRequest{}
			if specExecutionRequest.SlavePodRequest.Resources.Requests != nil {
				podRequest.Resources.Requests = &testkube.ResourceRequest{
					Cpu:    specExecutionRequest.SlavePodRequest.Resources.Requests.Cpu,
					Memory: specExecutionRequest.SlavePodRequest.Resources.Requests.Memory,
				}
			}

			if specExecutionRequest.SlavePodRequest.Resources.Limits != nil {
				podRequest.Resources.Limits = &testkube.ResourceRequest{
					Cpu:    specExecutionRequest.SlavePodRequest.Resources.Limits.Cpu,
					Memory: specExecutionRequest.SlavePodRequest.Resources.Limits.Memory,
				}
			}
		}

		podRequest.PodTemplate = specExecutionRequest.SlavePodRequest.PodTemplate
		podRequest.PodTemplateReference = specExecutionRequest.SlavePodRequest.PodTemplateReference
	}

	result := &testkube.ExecutionRequest{
		Name:                               specExecutionRequest.Name,
		TestSuiteName:                      specExecutionRequest.TestSuiteName,
		Number:                             specExecutionRequest.Number,
		ExecutionLabels:                    specExecutionRequest.ExecutionLabels,
		Namespace:                          specExecutionRequest.Namespace,
		IsVariablesFileUploaded:            specExecutionRequest.IsVariablesFileUploaded,
		VariablesFile:                      specExecutionRequest.VariablesFile,
		Variables:                          MergeVariablesAndParams(specExecutionRequest.Variables, nil),
		TestSecretUUID:                     specExecutionRequest.TestSecretUUID,
		TestSuiteSecretUUID:                specExecutionRequest.TestSuiteSecretUUID,
		Command:                            specExecutionRequest.Command,
		Args:                               specExecutionRequest.Args,
		ArgsMode:                           string(specExecutionRequest.ArgsMode),
		Image:                              specExecutionRequest.Image,
		ImagePullSecrets:                   MapImagePullSecrets(specExecutionRequest.ImagePullSecrets),
		Envs:                               specExecutionRequest.Envs,
		SecretEnvs:                         specExecutionRequest.SecretEnvs,
		Sync:                               specExecutionRequest.Sync,
		HttpProxy:                          specExecutionRequest.HttpProxy,
		HttpsProxy:                         specExecutionRequest.HttpsProxy,
		ActiveDeadlineSeconds:              specExecutionRequest.ActiveDeadlineSeconds,
		ArtifactRequest:                    artifactRequest,
		JobTemplate:                        specExecutionRequest.JobTemplate,
		JobTemplateReference:               specExecutionRequest.JobTemplateReference,
		CronJobTemplate:                    specExecutionRequest.CronJobTemplate,
		CronJobTemplateReference:           specExecutionRequest.CronJobTemplateReference,
		PreRunScript:                       specExecutionRequest.PreRunScript,
		PostRunScript:                      specExecutionRequest.PostRunScript,
		ExecutePostRunScriptBeforeScraping: specExecutionRequest.ExecutePostRunScriptBeforeScraping,
		SourceScripts:                      specExecutionRequest.SourceScripts,
		PvcTemplate:                        specExecutionRequest.PvcTemplate,
		PvcTemplateReference:               specExecutionRequest.PvcTemplateReference,
		ScraperTemplate:                    specExecutionRequest.ScraperTemplate,
		ScraperTemplateReference:           specExecutionRequest.ScraperTemplateReference,
		NegativeTest:                       specExecutionRequest.NegativeTest,
		EnvConfigMaps:                      MapEnvReferences(specExecutionRequest.EnvConfigMaps),
		EnvSecrets:                         MapEnvReferences(specExecutionRequest.EnvSecrets),
		SlavePodRequest:                    podRequest,
	}

	// Pro edition only (tcl protected code)
	return mappertcl.MapExecutionRequestFromSpec(specExecutionRequest, result)
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

// MapEnvReferences maps CRD to OpenAPI spec EnvReference
func MapEnvReferences(envs []testsv3.EnvReference) []testkube.EnvReference {
	if envs == nil {
		return nil
	}
	var res []testkube.EnvReference
	for _, env := range envs {
		res = append(res, testkube.EnvReference{
			Reference: &testkube.LocalObjectReference{
				Name: env.Name,
			},
			Mount:          env.Mount,
			MountPath:      env.MountPath,
			MapToVariables: env.MapToVariables,
		})
	}

	return res
}

// MapSpecToUpdate maps Test CRD spec to TestUpdateRequest
func MapSpecToUpdate(test *testsv3.Test) (request testkube.TestUpdateRequest) {
	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&test.Name,
			&request.Name,
		},
		{
			&test.Namespace,
			&request.Namespace,
		},
		{
			&test.Spec.Description,
			&request.Description,
		},
		{
			&test.Spec.Type_,
			&request.Type_,
		},
		{
			&test.Spec.Source,
			&request.Source,
		},
		{
			&test.Spec.Schedule,
			&request.Schedule,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	if test.Spec.Content != nil {
		content := MapSpecContentToUpdateContent(test.Spec.Content)
		request.Content = &content
	}

	if test.Spec.ExecutionRequest != nil {
		executionRequest := MapSpecExecutionRequestToExecutionUpdateRequest(test.Spec.ExecutionRequest)
		request.ExecutionRequest = &executionRequest
	}

	request.Labels = &test.Labels

	request.Uploads = &test.Spec.Uploads

	return request
}

// MapSpecContentToUpdateContent maps TestContent CRD spec to TestUpdateContent OpenAPI spec
func MapSpecContentToUpdateContent(testContent *testsv3.TestContent) (content *testkube.TestContentUpdate) {
	content = &testkube.TestContentUpdate{}

	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&testContent.Data,
			&content.Data,
		},
		{
			&testContent.Uri,
			&content.Uri,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	content.Type_ = (*string)(&testContent.Type_)

	if testContent.Repository != nil {
		repository := &testkube.RepositoryUpdate{}
		content.Repository = &repository

		var fields = []struct {
			source      *string
			destination **string
		}{
			{
				&testContent.Repository.Type_,
				&(*content.Repository).Type_,
			},
			{
				&testContent.Repository.Uri,
				&(*content.Repository).Uri,
			},
			{
				&testContent.Repository.Branch,
				&(*content.Repository).Branch,
			},
			{
				&testContent.Repository.Commit,
				&(*content.Repository).Commit,
			},
			{
				&testContent.Repository.Path,
				&(*content.Repository).Path,
			},
			{
				&testContent.Repository.WorkingDir,
				&(*content.Repository).WorkingDir,
			},
			{
				&testContent.Repository.CertificateSecret,
				&(*content.Repository).CertificateSecret,
			},
		}

		for _, field := range fields {
			*field.destination = field.source
		}

		(*content.Repository).AuthType = (*string)(&testContent.Repository.AuthType)

		if testContent.Repository.UsernameSecret != nil {
			secetRef := &testkube.SecretRef{
				Name: testContent.Repository.UsernameSecret.Name,
				Key:  testContent.Repository.UsernameSecret.Key,
			}

			(*content.Repository).TokenSecret = &secetRef
		}

		if testContent.Repository.TokenSecret != nil {
			secretRef := &testkube.SecretRef{
				Name: testContent.Repository.TokenSecret.Name,
				Key:  testContent.Repository.TokenSecret.Key,
			}

			(*content.Repository).TokenSecret = &secretRef
		}
	}

	return content
}

// MapSpecExecutionRequestToExecutionUpdateRequest maps ExecutionRequest CRD spec to ExecutionUpdateRequest OpenAPI spec to
func MapSpecExecutionRequestToExecutionUpdateRequest(
	request *testsv3.ExecutionRequest) (executionRequest *testkube.ExecutionUpdateRequest) {
	executionRequest = &testkube.ExecutionUpdateRequest{}

	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&request.Name,
			&executionRequest.Name,
		},
		{
			&request.TestSuiteName,
			&executionRequest.TestSuiteName,
		},
		{
			&request.Namespace,
			&executionRequest.Namespace,
		},
		{
			&request.VariablesFile,
			&executionRequest.VariablesFile,
		},
		{
			&request.TestSecretUUID,
			&executionRequest.TestSecretUUID,
		},
		{
			&request.TestSuiteSecretUUID,
			&executionRequest.TestSuiteSecretUUID,
		},
		{
			&request.HttpProxy,
			&executionRequest.HttpProxy,
		},
		{
			&request.HttpsProxy,
			&executionRequest.HttpsProxy,
		},
		{
			&request.Image,
			&executionRequest.Image,
		},
		{
			&request.JobTemplate,
			&executionRequest.JobTemplate,
		},
		{
			&request.JobTemplateReference,
			&executionRequest.JobTemplateReference,
		},
		{
			&request.PreRunScript,
			&executionRequest.PreRunScript,
		},
		{
			&request.PostRunScript,
			&executionRequest.PostRunScript,
		},
		{
			&request.CronJobTemplate,
			&executionRequest.CronJobTemplate,
		},
		{
			&request.CronJobTemplateReference,
			&executionRequest.CronJobTemplateReference,
		},
		{
			&request.PvcTemplate,
			&executionRequest.PvcTemplate,
		},
		{
			&request.PvcTemplateReference,
			&executionRequest.PvcTemplateReference,
		},
		{
			&request.ScraperTemplate,
			&executionRequest.ScraperTemplate,
		},
		{
			&request.ScraperTemplateReference,
			&executionRequest.ScraperTemplateReference,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	executionRequest.ArgsMode = (*string)(&request.ArgsMode)

	var slices = []struct {
		source      *map[string]string
		destination **map[string]string
	}{
		{
			&request.ExecutionLabels,
			&executionRequest.ExecutionLabels,
		},
		{
			&request.Envs,
			&executionRequest.Envs,
		},
		{
			&request.SecretEnvs,
			&executionRequest.SecretEnvs,
		},
	}

	for _, slice := range slices {
		*slice.destination = slice.source
	}

	executionRequest.Number = &request.Number
	executionRequest.Sync = &request.Sync
	executionRequest.NegativeTest = &request.NegativeTest
	executionRequest.ActiveDeadlineSeconds = &request.ActiveDeadlineSeconds
	executionRequest.Args = &request.Args
	executionRequest.Command = &request.Command

	vars := MergeVariablesAndParams(request.Variables, nil)
	executionRequest.Variables = &vars
	imagePullSecrets := MapImagePullSecrets(request.ImagePullSecrets)
	executionRequest.ImagePullSecrets = &imagePullSecrets
	envConfigMaps := MapEnvReferences(request.EnvConfigMaps)
	executionRequest.EnvConfigMaps = &envConfigMaps
	envSecrets := MapEnvReferences(request.EnvSecrets)
	executionRequest.EnvSecrets = &envSecrets
	executionRequest.ExecutePostRunScriptBeforeScraping = &request.ExecutePostRunScriptBeforeScraping
	executionRequest.SourceScripts = &request.SourceScripts

	// Pro edition only (tcl protected code)
	mappertcl.MapSpecExecutionRequestToExecutionUpdateRequest(request, executionRequest)

	if request.ArtifactRequest != nil {
		artifactRequest := &testkube.ArtifactUpdateRequest{
			StorageClassName:       &request.ArtifactRequest.StorageClassName,
			VolumeMountPath:        &request.ArtifactRequest.VolumeMountPath,
			Dirs:                   &request.ArtifactRequest.Dirs,
			Masks:                  &request.ArtifactRequest.Masks,
			StorageBucket:          &request.ArtifactRequest.StorageBucket,
			OmitFolderPerExecution: &request.ArtifactRequest.OmitFolderPerExecution,
			SharedBetweenPods:      &request.ArtifactRequest.SharedBetweenPods,
		}

		executionRequest.ArtifactRequest = &artifactRequest
	}

	if request.SlavePodRequest != nil {
		podRequest := &testkube.PodUpdateRequest{
			PodTemplate:          &request.SlavePodRequest.PodTemplate,
			PodTemplateReference: &request.SlavePodRequest.PodTemplateReference,
		}

		if request.SlavePodRequest.Resources != nil {
			resources := &testkube.PodResourcesUpdateRequest{}
			if request.SlavePodRequest.Resources.Requests != nil {
				requests := testkube.ResourceUpdateRequest{
					Cpu:    &request.SlavePodRequest.Resources.Requests.Cpu,
					Memory: &request.SlavePodRequest.Resources.Requests.Memory,
				}

				resources.Requests = &requests
			}

			if request.SlavePodRequest.Resources.Limits != nil {
				limits := testkube.ResourceUpdateRequest{
					Cpu:    &request.SlavePodRequest.Resources.Limits.Cpu,
					Memory: &request.SlavePodRequest.Resources.Limits.Memory,
				}

				resources.Limits = &limits
			}

			podRequest.Resources = &resources
		}

		executionRequest.SlavePodRequest = &podRequest
	}

	return executionRequest
}
