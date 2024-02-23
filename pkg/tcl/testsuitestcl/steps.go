// Copyright 2024 Kubeshop.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/master/licenses/TCL.txt

package testsuitestcl

import (
	v1 "github.com/kubeshop/testkube-operator/api/common/v1"
	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	testsuitestclop "github.com/kubeshop/testkube-operator/pkg/tcl/testsuitestcl"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MergeStepRequest inherits step request fields with execution request
func MergeStepRequest(stepRequest *testkube.TestSuiteStepExecutionRequest, executionRequest testkube.ExecutionRequest) testkube.ExecutionRequest {
	executionRequest.Name = setStringField(executionRequest.Name, stepRequest.Name)
	executionRequest.Namespace = setStringField(executionRequest.Namespace, stepRequest.Namespace)

	if stepRequest.ExecutionLabels != nil {
		executionRequest.ExecutionLabels = stepRequest.ExecutionLabels
	}

	if stepRequest.Variables != nil {
		// TODO test this well with both direct, configmaps and secret vars
		executionRequest.Variables = mergeVariables(executionRequest.Variables, stepRequest.Variables)
	}

	if len(stepRequest.Args) != 0 {
		if stepRequest.ArgsMode == string(testkube.ArgsModeTypeAppend) || stepRequest.ArgsMode == "" {
			executionRequest.Args = append(executionRequest.Args, stepRequest.Args...)
		}

		if stepRequest.ArgsMode == string(testkube.ArgsModeTypeOverride) || stepRequest.ArgsMode == string(testkube.ArgsModeTypeReplace) {
			executionRequest.Args = stepRequest.Args
		}
	}

	if stepRequest.Command != nil {
		executionRequest.Command = stepRequest.Command
	}
	executionRequest.Sync = stepRequest.Sync
	executionRequest.HttpProxy = setStringField(executionRequest.HttpProxy, stepRequest.HttpProxy)
	executionRequest.HttpsProxy = setStringField(executionRequest.HttpsProxy, stepRequest.HttpsProxy)
	executionRequest.CronJobTemplate = setStringField(executionRequest.CronJobTemplate, stepRequest.CronJobTemplate)
	executionRequest.CronJobTemplateReference = setStringField(executionRequest.CronJobTemplateReference, stepRequest.CronJobTemplateReference)
	executionRequest.JobTemplate = setStringField(executionRequest.JobTemplate, stepRequest.JobTemplate)
	executionRequest.JobTemplateReference = setStringField(executionRequest.JobTemplateReference, stepRequest.JobTemplateReference)
	executionRequest.ScraperTemplate = setStringField(executionRequest.ScraperTemplate, stepRequest.ScraperTemplate)
	executionRequest.ScraperTemplateReference = setStringField(executionRequest.ScraperTemplateReference, stepRequest.ScraperTemplateReference)
	executionRequest.PvcTemplate = setStringField(executionRequest.PvcTemplate, stepRequest.PvcTemplate)
	executionRequest.PvcTemplateReference = setStringField(executionRequest.PvcTemplate, stepRequest.PvcTemplateReference)

	if stepRequest.RunningContext != nil {
		executionRequest.RunningContext = &testkube.RunningContext{
			Type_:   string(stepRequest.RunningContext.Type_),
			Context: stepRequest.RunningContext.Context,
		}
	}

	return executionRequest
}

// HasStepsExecutionRequest checks if test suite has steps with execution requests
func HasStepsExecutionRequest(testSuite testsuitesv3.TestSuite) bool {
	for _, batch := range testSuite.Spec.Before {
		for _, step := range batch.Execute {
			if step.ExecutionRequest != nil {
				return true
			}
		}
	}
	for _, batch := range testSuite.Spec.Steps {
		for _, step := range batch.Execute {
			if step.ExecutionRequest != nil {
				return true
			}
		}
	}
	for _, batch := range testSuite.Spec.After {
		for _, step := range batch.Execute {
			if step.ExecutionRequest != nil {
				return true
			}
		}
	}
	return false
}

func setStringField(oldValue string, newValue string) string {
	if newValue != "" {
		return newValue
	}
	return oldValue
}

func mergeVariables(vars1 map[string]testkube.Variable, vars2 map[string]testkube.Variable) map[string]testkube.Variable {
	variables := map[string]testkube.Variable{}
	for k, v := range vars1 {
		variables[k] = v
	}

	for k, v := range vars2 {
		variables[k] = v
	}

	return variables
}

func MapTestStepExecutionRequestCRD(request *testkube.TestSuiteStepExecutionRequest) *testsuitestclop.TestSuiteStepExecutionRequest {
	if request == nil {
		return nil
	}

	variables := map[string]testsuitestclop.Variable{}
	for k, v := range request.Variables {
		variables[k] = testsuitestclop.Variable{
			Name:  v.Name,
			Value: v.Value,
			Type_: string(*v.Type_),
		}
	}

	var runningContext *v1.RunningContext
	if request.RunningContext != nil {
		runningContext = &v1.RunningContext{
			Type_:   v1.RunningContextType(request.RunningContext.Type_),
			Context: request.RunningContext.Context,
		}
	}

	return &testsuitestclop.TestSuiteStepExecutionRequest{
		Name:                     request.Name,
		ExecutionLabels:          request.ExecutionLabels,
		Namespace:                request.Namespace,
		Variables:                variables,
		Args:                     request.Args,
		ArgsMode:                 testsuitestclop.ArgsModeType(request.ArgsMode),
		Command:                  request.Command,
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

func MapTestStepExecutionRequestCRDToAPI(request *testsuitestclop.TestSuiteStepExecutionRequest) *testkube.TestSuiteStepExecutionRequest {
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
		Name:                     request.Name,
		ExecutionLabels:          request.ExecutionLabels,
		Namespace:                request.Namespace,
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
