// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package schedulertcl

import (
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewExecutionFromExecutionOptions creates new execution from execution options
func NewExecutionFromExecutionOptions(request testkube.ExecutionRequest, execution testkube.Execution) testkube.Execution {
	execution.ExecutionNamespace = request.ExecutionNamespace

	return execution
}

// GetExecuteOptions returns execute options
func GetExecuteOptions(sourceRequest *testkube.ExecutionRequest,
	destinationRequest testkube.ExecutionRequest) testkube.ExecutionRequest {
	if sourceRequest == nil {
		return destinationRequest
	}

	if destinationRequest.ExecutionNamespace == "" && sourceRequest.ExecutionNamespace != "" {
		destinationRequest.ExecutionNamespace = sourceRequest.ExecutionNamespace
	}

	if destinationRequest.ExecutionNamespace != "" {
		destinationRequest.Namespace = destinationRequest.ExecutionNamespace
	}

	return destinationRequest
}

// HasExecutionNamespace checks whether execution has execution namespace
func HasExecutionNamespace(request *testkube.ExecutionRequest) bool {
	return request.ExecutionNamespace != ""
}

// GetServiceAccountNamesFromConfig returns service account names from config
func GetServiceAccountNamesFromConfig(serviceAccountNames map[string]string, config string) map[string]string {
	if serviceAccountNames == nil {
		serviceAccountNames = make(map[string]string)
	}

	items := strings.Split(config, ",")
	for _, item := range items {
		elements := strings.Split(item, "=")
		if len(elements) != 2 {
			continue
		}

		serviceAccountNames[elements[0]] = elements[1]
	}

	return serviceAccountNames
}
