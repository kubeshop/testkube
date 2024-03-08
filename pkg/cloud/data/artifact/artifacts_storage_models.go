package artifact

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type ListFilesRequest struct {
	ExecutionID      string
	TestName         string
	TestSuiteName    string
	TestWorkflowName string
}

type ListFilesResponse struct {
	Artifacts []testkube.Artifact
}

type DownloadFileRequest struct {
	File             string
	ExecutionID      string
	TestName         string
	TestSuiteName    string
	TestWorkflowName string
}

type DownloadFileResponse struct {
	URL string
}
