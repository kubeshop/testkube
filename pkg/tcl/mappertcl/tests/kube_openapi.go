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

// MapExecutionRequestFromSpec maps CRD to OpenAPI spec ExecutionREquest
func MapExecutionRequestFromSpec(sourceRequest *testsv3.ExecutionRequest,
	destinationRequest *testkube.ExecutionRequest) *testkube.ExecutionRequest {
	if sourceRequest == nil || destinationRequest == nil {
		return destinationRequest
	}

	destinationRequest.ExecutionNamespace = sourceRequest.ExecutionNamespace
	return destinationRequest
}

// MapSpecExecutionRequestToExecutionUpdateRequest maps ExecutionRequest CRD spec to ExecutionUpdateRequest OpenAPI spec to
func MapSpecExecutionRequestToExecutionUpdateRequest(
	sourceRequest *testsv3.ExecutionRequest, destinationRequest *testkube.ExecutionUpdateRequest) {
	if sourceRequest == nil || destinationRequest == nil {
		return
	}

	destinationRequest.ExecutionNamespace = &sourceRequest.ExecutionNamespace
}
