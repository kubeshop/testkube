// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package tests

import (
	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapExecutionRequestToSpecExecutionRequest maps ExecutionRequest OpenAPI spec to ExecutionRequest CRD spec
func MapExecutionRequestToSpecExecutionRequest(sourceRequest *testkube.ExecutionRequest,
	destinationRequest *testsv3.ExecutionRequest) *testsv3.ExecutionRequest {
	if sourceRequest == nil || destinationRequest == nil {
		return destinationRequest
	}

	destinationRequest.ExecutionNamespace = sourceRequest.ExecutionNamespace
	return destinationRequest
}

// MapExecutionUpdateRequestToSpecExecutionRequest maps ExecutionUpdateRequest OpenAPI spec to ExecutionRequest CRD spec
func MapExecutionUpdateRequestToSpecExecutionRequest(sourceRequest *testkube.ExecutionUpdateRequest,
	destinationRequest *testsv3.ExecutionRequest) bool {
	if sourceRequest == nil || destinationRequest == nil {
		return true
	}

	if sourceRequest.ExecutionNamespace != nil {
		destinationRequest.ExecutionNamespace = *sourceRequest.ExecutionNamespace
		return false
	}

	return true
}
