package testsuiteexecutions

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsuiteexecutionv1 "github.com/kubeshop/testkube-operator/api/testsuiteexecution/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	mappertcl "github.com/kubeshop/testkube/pkg/tcl/mappertcl/testsuiteexecutions"
)

// MapCRDVariables maps variables between API and operator CRDs
func MapCRDVariables(in map[string]testkube.Variable) map[string]testsuiteexecutionv1.Variable {
	out := map[string]testsuiteexecutionv1.Variable{}
	for k, v := range in {
		variable := testsuiteexecutionv1.Variable{
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
func MapContentToSpecContent(content *testkube.TestContent) (specContent *testsuiteexecutionv1.TestContent) {
	if content == nil {
		return
	}

	var repository *testsuiteexecutionv1.Repository
	if content.Repository != nil {
		repository = &testsuiteexecutionv1.Repository{
			Type_:             content.Repository.Type_,
			Uri:               content.Repository.Uri,
			Branch:            content.Repository.Branch,
			Commit:            content.Repository.Commit,
			Path:              content.Repository.Path,
			WorkingDir:        content.Repository.WorkingDir,
			CertificateSecret: content.Repository.CertificateSecret,
			AuthType:          testsuiteexecutionv1.GitAuthType(content.Repository.AuthType),
		}

		if content.Repository.UsernameSecret != nil {
			repository.UsernameSecret = &testsuiteexecutionv1.SecretRef{
				Name: content.Repository.UsernameSecret.Name,
				Key:  content.Repository.UsernameSecret.Key,
			}
		}

		if content.Repository.TokenSecret != nil {
			repository.TokenSecret = &testsuiteexecutionv1.SecretRef{
				Name: content.Repository.TokenSecret.Name,
				Key:  content.Repository.TokenSecret.Key,
			}
		}
	}

	return &testsuiteexecutionv1.TestContent{
		Repository: repository,
		Data:       content.Data,
		Uri:        content.Uri,
		Type_:      testsuiteexecutionv1.TestContentType(content.Type_),
	}
}

// MapExecutionResultToCRD maps OpenAPI spec ExecutionResult to CRD ExecutionResult
func MapExecutionResultToCRD(result *testkube.ExecutionResult) *testsuiteexecutionv1.ExecutionResult {
	if result == nil {
		return nil
	}

	var status *testsuiteexecutionv1.ExecutionStatus
	if result.Status != nil {
		value := testsuiteexecutionv1.ExecutionStatus(*result.Status)
		status = &value
	}

	var steps []testsuiteexecutionv1.ExecutionStepResult
	for _, step := range result.Steps {
		var asserstions []testsuiteexecutionv1.AssertionResult
		for _, asserstion := range step.AssertionResults {
			asserstions = append(asserstions, testsuiteexecutionv1.AssertionResult{
				Name:         asserstion.Name,
				Status:       asserstion.Status,
				ErrorMessage: asserstion.ErrorMessage,
			})
		}

		steps = append(steps, testsuiteexecutionv1.ExecutionStepResult{
			Name:             step.Name,
			Duration:         step.Duration,
			Status:           step.Status,
			AssertionResults: asserstions,
		})
	}

	var reports *testsuiteexecutionv1.ExecutionResultReports
	if result.Reports != nil {
		reports = &testsuiteexecutionv1.ExecutionResultReports{
			Junit: result.Reports.Junit,
		}
	}

	return &testsuiteexecutionv1.ExecutionResult{
		Status:       status,
		ErrorMessage: result.ErrorMessage,
		Steps:        steps,
		Reports:      reports,
	}
}

// MapExecutionCRD maps OpenAPI spec Execution to CRD
func MapExecutionCRD(request *testkube.Execution) *testsuiteexecutionv1.Execution {
	if request == nil {
		return nil
	}

	var artifactRequest *testsuiteexecutionv1.ArtifactRequest
	if request.ArtifactRequest != nil {
		artifactRequest = &testsuiteexecutionv1.ArtifactRequest{
			StorageClassName:       request.ArtifactRequest.StorageClassName,
			VolumeMountPath:        request.ArtifactRequest.VolumeMountPath,
			Dirs:                   request.ArtifactRequest.Dirs,
			Masks:                  request.ArtifactRequest.Masks,
			StorageBucket:          request.ArtifactRequest.StorageBucket,
			OmitFolderPerExecution: request.ArtifactRequest.OmitFolderPerExecution,
			SharedBetweenPods:      request.ArtifactRequest.SharedBetweenPods,
		}
	}

	var runningContext *testsuiteexecutionv1.RunningContext
	if request.RunningContext != nil {
		runningContext = &testsuiteexecutionv1.RunningContext{
			Type_:   testsuiteexecutionv1.RunningContextType(request.RunningContext.Type_),
			Context: request.RunningContext.Context,
		}
	}

	var podRequest *testsuiteexecutionv1.PodRequest
	if request.SlavePodRequest != nil {
		podRequest = &testsuiteexecutionv1.PodRequest{}
		if request.SlavePodRequest.Resources != nil {
			podRequest.Resources = &testsuiteexecutionv1.PodResourcesRequest{}
			if request.SlavePodRequest.Resources.Requests != nil {
				podRequest.Resources.Requests = &testsuiteexecutionv1.ResourceRequest{
					Cpu:    request.SlavePodRequest.Resources.Requests.Cpu,
					Memory: request.SlavePodRequest.Resources.Requests.Memory,
				}
			}

			if request.SlavePodRequest.Resources.Limits != nil {
				podRequest.Resources.Limits = &testsuiteexecutionv1.ResourceRequest{
					Cpu:    request.SlavePodRequest.Resources.Limits.Cpu,
					Memory: request.SlavePodRequest.Resources.Limits.Memory,
				}
			}
		}

		podRequest.PodTemplate = request.SlavePodRequest.PodTemplate
		podRequest.PodTemplateReference = request.SlavePodRequest.PodTemplateReference
	}

	result := &testsuiteexecutionv1.Execution{
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
		ArgsMode:                           testsuiteexecutionv1.ArgsModeType(request.ArgsMode),
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
	}

	result.StartTime.Time = request.StartTime
	result.EndTime.Time = request.EndTime

	// Pro edition only (tcl protected code)
	return mappertcl.MapExecutionCRD(request, result)
}

func MapTestSuiteStepV2ToCRD(request *testkube.TestSuiteStepV2) *testsuiteexecutionv1.TestSuiteStepV2 {
	if request == nil {
		return nil
	}

	var execute *testsuiteexecutionv1.TestSuiteStepExecuteTestV2
	var delay *testsuiteexecutionv1.TestSuiteStepDelayV2

	if request.Execute != nil {
		execute = &testsuiteexecutionv1.TestSuiteStepExecuteTestV2{
			Name:      request.Execute.Name,
			Namespace: request.Execute.Namespace,
		}
	}

	if request.Delay != nil {
		delay = &testsuiteexecutionv1.TestSuiteStepDelayV2{
			Duration: request.Delay.Duration,
		}
	}

	return &testsuiteexecutionv1.TestSuiteStepV2{
		StopTestOnFailure: request.StopTestOnFailure,
		Execute:           execute,
		Delay:             delay,
	}
}

func MapTestSuiteBatchStepToCRD(request *testkube.TestSuiteBatchStep) *testsuiteexecutionv1.TestSuiteBatchStep {
	if request == nil {
		return nil
	}

	var steps []testsuiteexecutionv1.TestSuiteStep
	for _, step := range request.Execute {
		steps = append(steps, testsuiteexecutionv1.TestSuiteStep{
			Test:  step.Test,
			Delay: step.Delay,
		})
	}

	return &testsuiteexecutionv1.TestSuiteBatchStep{
		StopOnFailure: request.StopOnFailure,
		Execute:       steps,
	}
}

// MapAPIToCRD maps OpenAPI spec Execution to CRD TestSuiteExecutionStatus
func MapAPIToCRD(request *testkube.TestSuiteExecution, generation int64) testsuiteexecutionv1.TestSuiteExecutionStatus {
	var testSuite *testsuiteexecutionv1.ObjectRef
	if request.TestSuite != nil {
		testSuite = &testsuiteexecutionv1.ObjectRef{
			Name:      request.TestSuite.Name,
			Namespace: request.TestSuite.Namespace,
		}
	}

	var status *testsuiteexecutionv1.SuiteExecutionStatus
	if request.Status != nil {
		value := testsuiteexecutionv1.SuiteExecutionStatus(*request.Status)
		status = &value
	}

	var runningContext *testsuiteexecutionv1.RunningContext
	if request.RunningContext != nil {
		runningContext = &testsuiteexecutionv1.RunningContext{
			Type_:   testsuiteexecutionv1.RunningContextType(request.RunningContext.Type_),
			Context: request.RunningContext.Context,
		}
	}

	var stepResults []testsuiteexecutionv1.TestSuiteStepExecutionResultV2
	for _, stepResult := range request.StepResults {
		var test *testsuiteexecutionv1.ObjectRef
		if stepResult.Test != nil {
			test = &testsuiteexecutionv1.ObjectRef{
				Name:      stepResult.Test.Name,
				Namespace: stepResult.Test.Namespace,
			}
		}

		stepResults = append(stepResults, testsuiteexecutionv1.TestSuiteStepExecutionResultV2{
			Step:      MapTestSuiteStepV2ToCRD(stepResult.Step),
			Test:      test,
			Execution: MapExecutionCRD(stepResult.Execution),
		})
	}

	var executeStepResults []testsuiteexecutionv1.TestSuiteBatchStepExecutionResult
	for _, stepResult := range request.ExecuteStepResults {
		var steps []testsuiteexecutionv1.TestSuiteStepExecutionResult
		for _, step := range stepResult.Execute {
			var testSuiteStep *testsuiteexecutionv1.TestSuiteStep
			if step.Step != nil {
				testSuiteStep = &testsuiteexecutionv1.TestSuiteStep{
					Test:  step.Step.Test,
					Delay: step.Step.Delay,
				}
			}

			var test *testsuiteexecutionv1.ObjectRef
			if step.Test != nil {
				test = &testsuiteexecutionv1.ObjectRef{
					Name:      step.Test.Name,
					Namespace: step.Test.Namespace,
				}
			}

			steps = append(steps, testsuiteexecutionv1.TestSuiteStepExecutionResult{
				Step:      testSuiteStep,
				Test:      test,
				Execution: MapExecutionCRD(step.Execution),
			})
		}

		executeStepResults = append(executeStepResults, testsuiteexecutionv1.TestSuiteBatchStepExecutionResult{
			Step:      MapTestSuiteBatchStepToCRD(stepResult.Step),
			Execute:   steps,
			StartTime: metav1.Time{Time: stepResult.StartTime},
			EndTime:   metav1.Time{Time: stepResult.EndTime},
			Duration:  stepResult.Duration,
		})
	}

	result := testsuiteexecutionv1.TestSuiteExecutionStatus{
		Generation: generation,
		LatestExecution: &testsuiteexecutionv1.SuiteExecution{
			Id:                 request.Id,
			Name:               request.Name,
			TestSuite:          testSuite,
			Status:             status,
			Envs:               request.Envs,
			Variables:          MapCRDVariables(request.Variables),
			SecretUUID:         request.SecretUUID,
			Duration:           request.Duration,
			DurationMs:         request.DurationMs,
			StepResults:        stepResults,
			ExecuteStepResults: executeStepResults,
			Labels:             request.Labels,
			RunningContext:     runningContext,
		},
	}

	result.LatestExecution.StartTime.Time = request.StartTime
	result.LatestExecution.EndTime.Time = request.EndTime
	return result
}
