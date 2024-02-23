// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package schedulertcl

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
)

// NewExecutionFromExecutionOptions creates new execution from execution options
func NewExecutionFromExecutionOptions(options client.ExecuteOptions, execution testkube.Execution) testkube.Execution {
	execution.ExecutionNamespace = options.Request.ExecutionNamespace
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

	return destinationRequest
}
