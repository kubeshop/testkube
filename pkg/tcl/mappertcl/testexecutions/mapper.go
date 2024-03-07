// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testexecutions

import (
	testexecutionv1 "github.com/kubeshop/testkube-operator/api/testexecution/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapAPIToCRD maps OpenAPI spec Execution to CRD TestExecutionStatus
func MapAPIToCRD(sourceRequest *testkube.Execution,
	destinationRequest *testexecutionv1.TestExecutionStatus) *testexecutionv1.TestExecutionStatus {
	if sourceRequest == nil || destinationRequest == nil {
		return destinationRequest
	}

	if destinationRequest.LatestExecution != nil {
		destinationRequest.LatestExecution.ExecutionNamespace = sourceRequest.ExecutionNamespace
	}

	return destinationRequest
}
