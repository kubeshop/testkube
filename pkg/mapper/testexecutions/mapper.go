package testexecutions

import (
	testexecutionv1 "github.com/kubeshop/testkube-operator/apis/testexecution/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapAPIToCRD maps OpenAPI spec Execution to CRD TestExecutionStatus
func MapAPIToCRD(request testkube.Execution) testexecutionv1.TestExecutionStatus {
	result := testexecutionv1.TestExecutionStatus{
		LatestExecution: &testexecutionv1.Execution{
			Id:            request.Id,
			TestName:      request.TestName,
			TestSuiteName: request.TestSuiteName,
			TestNamespace: request.TestNamespace,
			TestType:      request.TestType,
			Name:          request.Name,
			Number:        request.Number,
			Envs:          request.Envs,
			Command:       request.Command,
			Args:          request.Args,
			ArgsMode:      request.ArgsMode,
			// Variables
			IsVariablesFileUploaded: request.IsVariablesFileUploaded,
			VariablesFile:           request.VariablesFile,
			TestSecretUUID:          request.TestSecretUUID,
			// Content
			Duration:   request.Duration,
			DurationMs: request.DurationMs,
			// ExecutionResult
			Labels:     request.Labels,
			Uploads:    request.Uploads,
			BucketName: request.BucketName,
			// ArtifactRequest
			PreRunScript:  request.PreRunScript,
			PostRunScript: request.PostRunScript,
			// RunningContext
			ContainerShell: request.ContainerShell,
		},
	}

	result.LatestExecution.StartTime.Time = request.StartTime
	result.LatestExecution.EndTime.Time = request.EndTime
	return result
}
