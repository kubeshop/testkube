package artifact

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type ListFilesRequest struct {
	ExecutionID   string
	TestName      string
	TestSuiteName string
}

type ListFilesResponse struct {
	Artifacts []testkube.Artifact
}

type DownloadFileRequest struct {
	File          string
	ExecutionID   string
	TestName      string
	TestSuiteName string
}

type DownloadFileResponse struct {
	URL string
}

type DownloadArchiveRequest struct {
	ExecutionID string
}

type DownloadArchiveResponse struct {
	URL string
}
