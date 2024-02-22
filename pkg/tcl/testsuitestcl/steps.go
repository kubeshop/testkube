// Copyright 2024 Kubeshop.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/master/licenses/TCL.txt

package testsuitestcl

import (
	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
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
			if step.TestSuiteStepExecutionRequest != nil {
				return true
			}
		}
	}
	for _, batch := range testSuite.Spec.Steps {
		for _, step := range batch.Execute {
			if step.TestSuiteStepExecutionRequest != nil {
				return true
			}
		}
	}
	for _, batch := range testSuite.Spec.After {
		for _, step := range batch.Execute {
			if step.TestSuiteStepExecutionRequest != nil {
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
