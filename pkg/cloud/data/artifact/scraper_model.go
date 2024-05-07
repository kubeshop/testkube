package artifact

import "github.com/kubeshop/testkube/pkg/cloud/data/executor"

const CmdScraperPutObjectSignedURL executor.Command = "put_object_signed_url"

type PutObjectSignedURLRequest struct {
	Object           string `json:"object"`
	ContentType      string `json:"contentType,omitempty"`
	ExecutionID      string `json:"executionId"`
	TestName         string `json:"testName"`
	TestSuiteName    string `json:"testSuiteName"`
	TestWorkflowName string `json:"testWorkflowName"`
}

type PutObjectSignedURLResponse struct {
	URL string
}
