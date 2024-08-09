package testworkflow

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

var _ testworkflow.OutputRepository = (*CloudOutputRepository)(nil)

type CloudOutputRepository struct {
	executor   executor.Executor
	httpClient *http.Client
}

type Option func(*CloudOutputRepository)

func WithSkipVerify() Option {
	return func(r *CloudOutputRepository) {
		transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		r.httpClient.Transport = transport
	}
}

func NewCloudOutputRepository(client cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey, runnerId string, opts ...Option) *CloudOutputRepository {
	r := &CloudOutputRepository{executor: executor.NewCloudGRPCExecutor(client, grpcConn, apiKey, runnerId), httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// PresignSaveLog builds presigned storage URL to save the output in Cloud
func (r *CloudOutputRepository) PresignSaveLog(ctx context.Context, id, workflowName string) (string, error) {
	req := OutputPresignSaveLogRequest{ID: id, WorkflowName: workflowName}
	process := func(v OutputPresignSaveLogResponse) string {
		return v.URL
	}
	return pass(r.executor, ctx, req, process)
}

// PresignReadLog builds presigned storage URL to read the output from Cloud
func (r *CloudOutputRepository) PresignReadLog(ctx context.Context, id, workflowName string) (string, error) {
	req := OutputPresignReadLogRequest{ID: id, WorkflowName: workflowName}
	process := func(v OutputPresignReadLogResponse) string {
		return v.URL
	}
	return pass(r.executor, ctx, req, process)
}

// SaveLog streams the output from the workflow to Cloud
func (r *CloudOutputRepository) SaveLog(ctx context.Context, id, workflowName string, reader io.Reader) error {
	url, err := r.PresignSaveLog(ctx, id, workflowName)
	if err != nil {
		return err
	}
	// FIXME: It should stream instead
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(data))
	req.Header.Add("Content-Type", "application/octet-stream")
	if err != nil {
		return err
	}
	res, err := r.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to save file in cloud storage")
	}
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("error saving file with presigned url: expected 200 OK response code, got %d", res.StatusCode)
	}
	return nil
}

// ReadLog streams the output from Cloud
func (r *CloudOutputRepository) ReadLog(ctx context.Context, id, workflowName string) (io.Reader, error) {
	url, err := r.PresignReadLog(ctx, id, workflowName)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := r.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file from cloud storage")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("error getting file from presigned url: expected 200 OK response code, got %d", res.StatusCode)
	}
	return res.Body, nil
}

// HasLog checks if there is an output in Cloud
func (r *CloudOutputRepository) HasLog(ctx context.Context, id, workflowName string) (bool, error) {
	req := OutputHasLogRequest{ID: id, WorkflowName: workflowName}
	process := func(v OutputHasLogResponse) bool {
		return v.Has
	}
	return pass(r.executor, ctx, req, process)
}

// DeleteByTestWorkflow deletes execution results by workflow
func (r *CloudOutputRepository) DeleteOutputByTestWorkflow(ctx context.Context, workflowName string) (err error) {
	req := ExecutionDeleteOutputByWorkflowRequest{WorkflowName: workflowName}
	return passNoContent(r.executor, ctx, req)
}

func (r *CloudOutputRepository) DeleteOutputForTestWorkflows(ctx context.Context, workflowNames []string) (err error) {
	req := ExecutionDeleteOutputForTestWorkflowsRequest{WorkflowNames: workflowNames}
	return passNoContent(r.executor, ctx, req)
}
