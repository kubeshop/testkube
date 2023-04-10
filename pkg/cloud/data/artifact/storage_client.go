package artifact

import (
	context "context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/storage"
)

type CloudStorageClient struct {
	executor executor.Executor
}

func NewCloudStorageClient(cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudStorageClient {
	return &CloudStorageClient{executor: executor.NewCloudGRPCExecutor(cloudClient, grpcConn, apiKey)}
}

func (c *CloudStorageClient) ListFiles(ctx context.Context, executionID, testName, testSuiteName string) ([]testkube.Artifact, error) {
	req := ListFilesRequest{
		ExecutionID:   executionID,
		TestName:      testName,
		TestSuiteName: testSuiteName,
	}
	response, err := c.executor.Execute(ctx, CmdArtifactsListFiles, req)
	if err != nil {
		return nil, err
	}
	var commandResponse ListFilesResponse
	if err = json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}

	return commandResponse.Artifacts, nil
}

func (c *CloudStorageClient) DownloadFile(ctx context.Context, file, executionID, testName, testSuiteName string) (io.Reader, error) {
	req := DownloadFileRequest{
		File:          file,
		ExecutionID:   executionID,
		TestName:      testName,
		TestSuiteName: testSuiteName,
	}
	response, err := c.executor.Execute(ctx, CmdArtifactsDownloadFile, req)
	if err != nil {
		return nil, err
	}
	var commandResponse DownloadFileResponse
	if err = json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}

	data, err := c.getObject(ctx, commandResponse.URL)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *CloudStorageClient) DownloadArchive(ctx context.Context, executionID string, masks []string) (io.Reader, error) {
	req := DownloadArchiveRequest{
		ExecutionID: executionID,
		Masks:       masks,
	}
	response, err := c.executor.Execute(ctx, CmdArtifactsDownloadArchive, req)
	if err != nil {
		return nil, err
	}
	var commandResponse DownloadArchiveResponse
	if err = json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}

	data, err := c.getObject(ctx, commandResponse.URL)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *CloudStorageClient) getObject(ctx context.Context, url string) (io.Reader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file from cloud storage")
	}
	if rsp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("error getting file from presigned url: expected 200 OK response code, got %d", rsp.StatusCode)
	}

	return rsp.Body, nil
}

var _ storage.ArtifactsStorage = (*CloudStorageClient)(nil)
