package testsuites

import (
	corev1 "k8s.io/api/core/v1"

	commonv1 "github.com/kubeshop/testkube-operator/api/common/v1"
	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapTestSuiteListKubeToAPI maps TestSuiteList CRD to list of OpenAPI spec TestSuite
func MapTestSuiteListKubeToAPI(cr testsuitesv3.TestSuiteList) (tests []testkube.TestSuite) {
	tests = make([]testkube.TestSuite, len(cr.Items))
	for i, item := range cr.Items {
		tests[i] = MapCRToAPI(item)
	}

	return
}

// MapCRToAPI maps TestSuite CRD to OpenAPI spec TestSuite
func MapCRToAPI(cr testsuitesv3.TestSuite) (test testkube.TestSuite) {
	test.Name = cr.Name
	test.Namespace = cr.Namespace
	var batches = []struct {
		source *[]testsuitesv3.TestSuiteBatchStep
		dest   *[]testkube.TestSuiteBatchStep
	}{
		{
			source: &cr.Spec.Before,
			dest:   &test.Before,
		},
		{
			source: &cr.Spec.Steps,
			dest:   &test.Steps,
		},
		{
			source: &cr.Spec.After,
			dest:   &test.After,
		},
	}

	for i := range batches {
		for _, b := range *batches[i].source {
			steps := make([]testkube.TestSuiteStep, len(b.Execute))
			for j := range b.Execute {
				steps[j] = mapCRStepToAPI(b.Execute[j])
			}

			var downloadArtifacts *testkube.DownloadArtifactOptions
			if b.DownloadArtifacts != nil {
				downloadArtifacts = &testkube.DownloadArtifactOptions{
					AllPreviousSteps:    b.DownloadArtifacts.AllPreviousSteps,
					PreviousStepNumbers: b.DownloadArtifacts.PreviousStepNumbers,
					PreviousTestNames:   b.DownloadArtifacts.PreviousTestNames,
				}
			}

			*batches[i].dest = append(*batches[i].dest, testkube.TestSuiteBatchStep{
				StopOnFailure:     b.StopOnFailure,
				Execute:           steps,
				DownloadArtifacts: downloadArtifacts,
			})
		}
	}

	test.Description = cr.Spec.Description
	test.Repeats = int32(cr.Spec.Repeats)
	test.Labels = cr.Labels
	test.Schedule = cr.Spec.Schedule
	test.Created = cr.CreationTimestamp.Time
	test.ExecutionRequest = MapExecutionRequestFromSpec(cr.Spec.ExecutionRequest)
	test.Status = MapStatusFromSpec(cr.Status)
	return
}

// mapCRStepToAPI maps CRD TestSuiteStepSpec to OpenAPI spec TestSuiteStep
func mapCRStepToAPI(crstep testsuitesv3.TestSuiteStepSpec) (teststep testkube.TestSuiteStep) {

	switch true {
	case crstep.Test != "":
		teststep = testkube.TestSuiteStep{
			Test:             crstep.Test,
			ExecutionRequest: MapTestStepExecutionRequestCRDToAPI(crstep.ExecutionRequest),
		}

	case crstep.Delay.Duration != 0:
		teststep = testkube.TestSuiteStep{
			Delay: crstep.Delay.Duration.String(),
		}
	}

	return
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
// TODO if we could merge operator into testkube repository we would get rid of those mappings
func MapCRDVariables(in map[string]testkube.Variable) map[string]testsuitesv3.Variable {
	out := map[string]testsuitesv3.Variable{}
	for k, v := range in {
		variable := testsuitesv3.Variable{
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

func MergeVariablesAndParams(variables map[string]testsuitesv3.Variable, params map[string]string) map[string]testkube.Variable {
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

// MapExecutionRequestFromSpec maps CRD to OpenAPI spec ExecutionRequest
func MapExecutionRequestFromSpec(specExecutionRequest *testsuitesv3.TestSuiteExecutionRequest) *testkube.TestSuiteExecutionRequest {
	if specExecutionRequest == nil {
		return nil
	}

	return &testkube.TestSuiteExecutionRequest{
		Name:                     specExecutionRequest.Name,
		Labels:                   specExecutionRequest.Labels,
		ExecutionLabels:          specExecutionRequest.ExecutionLabels,
		Namespace:                specExecutionRequest.Namespace,
		Variables:                MergeVariablesAndParams(specExecutionRequest.Variables, nil),
		SecretUUID:               specExecutionRequest.SecretUUID,
		Sync:                     specExecutionRequest.Sync,
		HttpProxy:                specExecutionRequest.HttpProxy,
		HttpsProxy:               specExecutionRequest.HttpsProxy,
		Timeout:                  specExecutionRequest.Timeout,
		JobTemplate:              specExecutionRequest.JobTemplate,
		JobTemplateReference:     specExecutionRequest.JobTemplateReference,
		CronJobTemplate:          specExecutionRequest.CronJobTemplate,
		CronJobTemplateReference: specExecutionRequest.CronJobTemplateReference,
		PvcTemplate:              specExecutionRequest.PvcTemplate,
		PvcTemplateReference:     specExecutionRequest.PvcTemplateReference,
		ScraperTemplate:          specExecutionRequest.ScraperTemplate,
		ScraperTemplateReference: specExecutionRequest.ScraperTemplateReference,
	}
}

// MapStatusFromSpec maps CRD to OpenAPI spec TestSuiteStatus
func MapStatusFromSpec(specStatus testsuitesv3.TestSuiteStatus) *testkube.TestSuiteStatus {
	if specStatus.LatestExecution == nil {
		return nil
	}

	return &testkube.TestSuiteStatus{
		LatestExecution: &testkube.TestSuiteExecutionCore{
			Id:        specStatus.LatestExecution.Id,
			Status:    (*testkube.TestSuiteExecutionStatus)(specStatus.LatestExecution.Status),
			StartTime: specStatus.LatestExecution.StartTime.Time,
			EndTime:   specStatus.LatestExecution.EndTime.Time,
		},
	}
}

// MapTestSuiteTestCRDToUpdateRequest maps TestSuite CRD spec to TestSuiteUpdateRequest OpenAPI spec
func MapTestSuiteTestCRDToUpdateRequest(testSuite *testsuitesv3.TestSuite) (request testkube.TestSuiteUpdateRequest) {
	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&testSuite.Name,
			&request.Name,
		},
		{
			&testSuite.Namespace,
			&request.Namespace,
		},
		{
			&testSuite.Spec.Description,
			&request.Description,
		},
		{
			&testSuite.Spec.Schedule,
			&request.Schedule,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	before := mapCRDToTestBatchSteps(testSuite.Spec.Before)
	request.Before = &before

	steps := mapCRDToTestBatchSteps(testSuite.Spec.Steps)
	request.Steps = &steps

	after := mapCRDToTestBatchSteps(testSuite.Spec.After)
	request.After = &after

	request.Labels = &testSuite.Labels

	repeats := int32(testSuite.Spec.Repeats)
	request.Repeats = &repeats

	if testSuite.Spec.ExecutionRequest != nil {
		value := MapSpecExecutionRequestToExecutionUpdateRequest(testSuite.Spec.ExecutionRequest)
		request.ExecutionRequest = &value
	}

	return request
}

func mapCRDToTestBatchSteps(in []testsuitesv3.TestSuiteBatchStep) (batches []testkube.TestSuiteBatchStep) {
	for _, batch := range in {
		steps := make([]testkube.TestSuiteStep, len(batch.Execute))
		for i := range batch.Execute {
			steps[i] = mapCRStepToAPI(batch.Execute[i])
		}

		batches = append(batches, testkube.TestSuiteBatchStep{
			StopOnFailure: batch.StopOnFailure,
			Execute:       steps,
		})
	}

	return batches
}

// MapSpecExecutionRequestToExecutionUpdateRequest maps ExecutionRequest CRD spec to ExecutionUpdateRequest OpenAPI spec
func MapSpecExecutionRequestToExecutionUpdateRequest(request *testsuitesv3.TestSuiteExecutionRequest) (executionRequest *testkube.TestSuiteExecutionUpdateRequest) {
	executionRequest = &testkube.TestSuiteExecutionUpdateRequest{}

	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&request.Name,
			&executionRequest.Name,
		},
		{
			&request.Namespace,
			&executionRequest.Namespace,
		},
		{
			&request.SecretUUID,
			&executionRequest.SecretUUID,
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
			&request.JobTemplate,
			&executionRequest.JobTemplate,
		},
		{
			&request.JobTemplateReference,
			&executionRequest.JobTemplateReference,
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

	executionRequest.Labels = &request.Labels

	executionRequest.ExecutionLabels = &request.ExecutionLabels

	executionRequest.Sync = &request.Sync

	executionRequest.Timeout = &request.Timeout

	vars := MergeVariablesAndParams(request.Variables, nil)
	executionRequest.Variables = &vars

	return executionRequest
}

func MapTestStepExecutionRequestCRDToAPI(request *testsuitesv3.TestSuiteStepExecutionRequest) *testkube.TestSuiteStepExecutionRequest {
	if request == nil {
		return nil
	}
	variables := map[string]testkube.Variable{}
	for k, v := range request.Variables {
		varType := testkube.VariableType(v.Type_)
		variables[k] = testkube.Variable{
			Name:  v.Name,
			Value: v.Value,
			Type_: &varType,
		}
	}

	var runningContext *testkube.RunningContext

	if request.RunningContext != nil {
		runningContext = &testkube.RunningContext{
			Type_:   string(request.RunningContext.Type_),
			Context: request.RunningContext.Context,
		}
	}

	argsMode := ""
	if request.ArgsMode != "" {
		argsMode = string(request.ArgsMode)
	}

	return &testkube.TestSuiteStepExecutionRequest{
		ExecutionLabels:          request.ExecutionLabels,
		Variables:                variables,
		Command:                  request.Command,
		Args:                     request.Args,
		ArgsMode:                 argsMode,
		Sync:                     request.Sync,
		HttpProxy:                request.HttpProxy,
		HttpsProxy:               request.HttpsProxy,
		NegativeTest:             request.NegativeTest,
		JobTemplate:              request.JobTemplate,
		JobTemplateReference:     request.JobTemplateReference,
		CronJobTemplate:          request.CronJobTemplate,
		CronJobTemplateReference: request.CronJobTemplateReference,
		ScraperTemplate:          request.ScraperTemplate,
		ScraperTemplateReference: request.ScraperTemplateReference,
		PvcTemplate:              request.PvcTemplate,
		PvcTemplateReference:     request.PvcTemplateReference,
		RunningContext:           runningContext,
	}
}
