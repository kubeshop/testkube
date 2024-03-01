// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testsuiteexecutions

import (
	testsuiteexecutionv1 "github.com/kubeshop/testkube-operator/api/testsuiteexecution/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapExecutionCRD maps OpenAPI spec Execution to CRD
func MapExecutionCRD(sourceRequest *testkube.Execution,
	destinationRequest *testsuiteexecutionv1.Execution) *testsuiteexecutionv1.Execution {
	if sourceRequest == nil || destinationRequest == nil {
		return destinationRequest
	}

	destinationRequest.ExecutionNamespace = sourceRequest.ExecutionNamespace
	return destinationRequest
}
