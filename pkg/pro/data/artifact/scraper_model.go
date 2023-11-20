package artifact

import "github.com/kubeshop/testkube/pkg/cloud/data/executor"

const (
	CmdScraperPutObjectSignedURL executor.Command = "put_object_signed_url"
)

type PutObjectSignedURLRequest struct {
	Object        string `json:"object"`
	ExecutionID   string `json:"executionId"`
	TestName      string `json:"testName"`
	TestSuiteName string `json:"testSuiteName"`
}

type PutObjectSignedURLResponse struct {
	URL string
}
