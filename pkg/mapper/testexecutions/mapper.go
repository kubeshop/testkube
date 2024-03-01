package testexecutions

import (
	corev1 "k8s.io/api/core/v1"

	testexecutionv1 "github.com/kubeshop/testkube-operator/api/testexecution/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	mappertcl "github.com/kubeshop/testkube/pkg/tcl/mappertcl/testexecutions"
)

// MapCRDVariables maps variables between API and operator CRDs
func MapCRDVariables(in map[string]testkube.Variable) map[string]testexecutionv1.Variable {
	out := map[string]testexecutionv1.Variable{}
	for k, v := range in {
		variable := testexecutionv1.Variable{
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
func MapContentToSpecContent(content *testkube.TestContent) (specContent *testexecutionv1.TestContent) {
	if content == nil {
		return
	}

	var repository *testexecutionv1.Repository
	if content.Repository != nil {
		repository = &testexecutionv1.Repository{
			Type_:             content.Repository.Type_,
			Uri:               content.Repository.Uri,
			Branch:            content.Repository.Branch,
			Commit:            content.Repository.Commit,
			Path:              content.Repository.Path,
			WorkingDir:        content.Repository.WorkingDir,
			CertificateSecret: content.Repository.CertificateSecret,
			AuthType:          testexecutionv1.GitAuthType(content.Repository.AuthType),
		}

		if content.Repository.UsernameSecret != nil {
			repository.UsernameSecret = &testexecutionv1.SecretRef{
				Name: content.Repository.UsernameSecret.Name,
				Key:  content.Repository.UsernameSecret.Key,
			}
		}

		if content.Repository.TokenSecret != nil {
			repository.TokenSecret = &testexecutionv1.SecretRef{
				Name: content.Repository.TokenSecret.Name,
				Key:  content.Repository.TokenSecret.Key,
			}
		}
	}

	return &testexecutionv1.TestContent{
		Repository: repository,
		Data:       content.Data,
		Uri:        content.Uri,
		Type_:      testexecutionv1.TestContentType(content.Type_),
	}
}

// MapExecutionResultToCRD maps OpenAPI spec ExecutionResult to CRD ExecutionResult
func MapExecutionResultToCRD(result *testkube.ExecutionResult) *testexecutionv1.ExecutionResult {
	if result == nil {
		return nil
	}

	var status *testexecutionv1.ExecutionStatus
	if result.Status != nil {
		value := testexecutionv1.ExecutionStatus(*result.Status)
		status = &value
	}

	var steps []testexecutionv1.ExecutionStepResult
	for _, step := range result.Steps {
		var asserstions []testexecutionv1.AssertionResult
		for _, asserstion := range step.AssertionResults {
			asserstions = append(asserstions, testexecutionv1.AssertionResult{
				Name:         asserstion.Name,
				Status:       asserstion.Status,
				ErrorMessage: asserstion.ErrorMessage,
			})
		}

		steps = append(steps, testexecutionv1.ExecutionStepResult{
			Name:             step.Name,
			Duration:         step.Duration,
			Status:           step.Status,
			AssertionResults: asserstions,
		})
	}

	var reports *testexecutionv1.ExecutionResultReports
	if result.Reports != nil {
		reports = &testexecutionv1.ExecutionResultReports{
			Junit: result.Reports.Junit,
		}
	}

	return &testexecutionv1.ExecutionResult{
		Status:       status,
		ErrorMessage: result.ErrorMessage,
		Steps:        steps,
		Reports:      reports,
	}
}

// MapAPIToCRD maps OpenAPI spec Execution to CRD TestExecutionStatus
func MapAPIToCRD(request *testkube.Execution, generation int64) testexecutionv1.TestExecutionStatus {
	var artifactRequest *testexecutionv1.ArtifactRequest
	if request.ArtifactRequest != nil {
		artifactRequest = &testexecutionv1.ArtifactRequest{
			StorageClassName:       request.ArtifactRequest.StorageClassName,
			VolumeMountPath:        request.ArtifactRequest.VolumeMountPath,
			Dirs:                   request.ArtifactRequest.Dirs,
			Masks:                  request.ArtifactRequest.Masks,
			StorageBucket:          request.ArtifactRequest.StorageBucket,
			OmitFolderPerExecution: request.ArtifactRequest.OmitFolderPerExecution,
			SharedBetweenPods:      request.ArtifactRequest.SharedBetweenPods,
		}
	}

	var runningContext *testexecutionv1.RunningContext
	if request.RunningContext != nil {
		runningContext = &testexecutionv1.RunningContext{
			Type_:   testexecutionv1.RunningContextType(request.RunningContext.Type_),
			Context: request.RunningContext.Context,
		}
	}

	var podRequest *testexecutionv1.PodRequest
	if request.SlavePodRequest != nil {
		podRequest = &testexecutionv1.PodRequest{}
		if request.SlavePodRequest.Resources != nil {
			podRequest.Resources = &testexecutionv1.PodResourcesRequest{}
			if request.SlavePodRequest.Resources.Requests != nil {
				podRequest.Resources.Requests = &testexecutionv1.ResourceRequest{
					Cpu:    request.SlavePodRequest.Resources.Requests.Cpu,
					Memory: request.SlavePodRequest.Resources.Requests.Memory,
				}
			}

			if request.SlavePodRequest.Resources.Limits != nil {
				podRequest.Resources.Limits = &testexecutionv1.ResourceRequest{
					Cpu:    request.SlavePodRequest.Resources.Limits.Cpu,
					Memory: request.SlavePodRequest.Resources.Limits.Memory,
				}
			}
		}

		podRequest.PodTemplate = request.SlavePodRequest.PodTemplate
		podRequest.PodTemplateReference = request.SlavePodRequest.PodTemplateReference
	}

	result := testexecutionv1.TestExecutionStatus{
		Generation: generation,
		LatestExecution: &testexecutionv1.Execution{
			Id:                                 request.Id,
			TestName:                           request.TestName,
			TestSuiteName:                      request.TestSuiteName,
			TestNamespace:                      request.TestNamespace,
			TestType:                           request.TestType,
			Name:                               request.Name,
			Number:                             request.Number,
			Envs:                               request.Envs,
			Command:                            request.Command,
			Args:                               request.Args,
			ArgsMode:                           testexecutionv1.ArgsModeType(request.ArgsMode),
			Variables:                          MapCRDVariables(request.Variables),
			IsVariablesFileUploaded:            request.IsVariablesFileUploaded,
			VariablesFile:                      request.VariablesFile,
			TestSecretUUID:                     request.TestSecretUUID,
			Content:                            MapContentToSpecContent(request.Content),
			Duration:                           request.Duration,
			DurationMs:                         request.DurationMs,
			ExecutionResult:                    MapExecutionResultToCRD(request.ExecutionResult),
			Labels:                             request.Labels,
			Uploads:                            request.Uploads,
			BucketName:                         request.BucketName,
			ArtifactRequest:                    artifactRequest,
			PreRunScript:                       request.PreRunScript,
			PostRunScript:                      request.PostRunScript,
			ExecutePostRunScriptBeforeScraping: request.ExecutePostRunScriptBeforeScraping,
			SourceScripts:                      request.SourceScripts,
			RunningContext:                     runningContext,
			ContainerShell:                     request.ContainerShell,
			SlavePodRequest:                    podRequest,
		},
	}

	result.LatestExecution.StartTime.Time = request.StartTime
	result.LatestExecution.EndTime.Time = request.EndTime

	// Pro edition only (tcl protected code)
	return *mappertcl.MapAPIToCRD(request, &result)
}
