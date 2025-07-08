package controlplaneclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/bufferedstream"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/log"
)

type ExecutionClient interface {
	GetExecution(ctx context.Context, environmentId, executionId string) (*testkube.TestWorkflowExecution, error)
	SaveExecutionLogsGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName string) (string, error)
	SaveExecutionLogs(ctx context.Context, environmentId, executionId, legacyWorkflowName string, buffer io.Reader) error
	UpdateExecutionOutput(ctx context.Context, environmentId, executionId string, output []testkube.TestWorkflowOutput) error
	UpdateExecutionResult(ctx context.Context, environmentId, executionId string, result *testkube.TestWorkflowResult) error
	FinishExecutionResult(ctx context.Context, environmentId, executionId string, result *testkube.TestWorkflowResult) error
	InitExecution(ctx context.Context, environmentId, executionId string, signature []testkube.TestWorkflowSignature, namespace string) error
}

func (c *client) GetExecution(ctx context.Context, environmentId, executionId string) (*testkube.TestWorkflowExecution, error) {
	if c.IsLegacy() {
		return c.legacyGetExecution(ctx, environmentId, executionId)
	}
	req := &cloud.GetExecutionRequest{Id: executionId}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetExecution, req)
	if err != nil {
		return nil, err
	}
	var execution testkube.TestWorkflowExecution
	err = json.Unmarshal(res.Execution, &execution)
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// Deprecated
func (c *client) legacyGetExecution(ctx context.Context, environmentId, executionId string) (*testkube.TestWorkflowExecution, error) {
	jsonPayload, err := json.Marshal(cloudtestworkflow.ExecutionGetRequest{ID: executionId})
	if err != nil {
		return nil, err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return nil, err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowExecutionGet),
		Payload: &s,
	}
	cmdResponse, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.Call, &req)
	if err != nil {
		return nil, err
	}
	var response cloudtestworkflow.ExecutionGetResponse
	err = json.Unmarshal(cmdResponse.Response, &response)
	return &response.WorkflowExecution, err
}

func (c *client) SaveExecutionLogsGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName string) (string, error) {
	log.DefaultLogger.Debugw("grpc.SaveExecutionLogsGetPresignedURL", "id", executionId)
	if c.IsLegacy() {
		return c.legacySaveExecutionLogsGetPresignedURL(ctx, environmentId, executionId, legacyWorkflowName)
	}
	req := &cloud.SaveExecutionLogsPresignedRequest{Id: executionId}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.SaveExecutionLogsPresigned, req)
	if err != nil {
		return "", err
	}
	return res.Url, nil
}

// Deprecated
func (c *client) legacySaveExecutionLogsGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName string) (string, error) {
	jsonPayload, err := json.Marshal(cloudtestworkflow.OutputPresignSaveLogRequest{
		ID:           executionId,
		WorkflowName: legacyWorkflowName,
	})
	if err != nil {
		return "", err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return "", err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowOutputPresignSaveLog),
		Payload: &s,
	}
	cmdResponse, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.Call, &req)
	if err != nil {
		return "", err
	}
	var response cloudtestworkflow.OutputPresignSaveLogResponse
	err = json.Unmarshal(cmdResponse.Response, &response)
	return response.URL, err
}

func (c *client) SaveExecutionLogs(ctx context.Context, environmentId, executionId, legacyWorkflowName string, reader io.Reader) error {
	log.DefaultLogger.Debugw("grpc.SaveExecutionLogs", "id", executionId)
	// TODO: consider how to choose the temp dir
	buffer, err := bufferedstream.NewBufferedStream("", "log", reader)
	if err != nil {
		return err
	}
	bufferLen := buffer.Len()
	body := buffer.(io.Reader)
	if bufferLen == 0 {
		// http.Request won't send Content-Length: 0, if the body is non-nil
		buffer.Cleanup()
		body = http.NoBody
	} else {
		defer buffer.Cleanup()
	}
	url, err := c.SaveExecutionLogsGetPresignedURL(ctx, environmentId, executionId, legacyWorkflowName)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, body)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/octet-stream")
	req.ContentLength = int64(bufferLen)
	httpClient := http.DefaultClient
	if c.opts.StorageSkipVerify {
		transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		httpClient.Transport = transport
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to save file in cloud storage")
	}
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("error saving file with presigned url: expected 200 OK response code, got %d", res.StatusCode)
	}
	return nil
}

// TODO: Create AppendExecutionOutput (and maybe ResetExecutionOutput?) instead
func (c *client) UpdateExecutionOutput(ctx context.Context, environmentId, executionId string, output []testkube.TestWorkflowOutput) error {
	if c.IsLegacy() {
		return c.legacyUpdateExecutionOutput(ctx, environmentId, executionId, output)
	}
	req := &cloud.UpdateExecutionOutputRequest{
		Id: executionId,
		Output: common.MapSlice(output, func(t testkube.TestWorkflowOutput) *cloud.ExecutionOutput {
			v, _ := json.Marshal(t.Value)
			return &cloud.ExecutionOutput{Ref: t.Ref, Name: t.Name, Value: v}
		}),
	}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.UpdateExecutionOutput, req)
	return err
}

func (c *client) UpdateExecutionResult(ctx context.Context, environmentId, executionId string, result *testkube.TestWorkflowResult) error {
	log.DefaultLogger.Debugw("grpc.UpdateExecutionResult", "id", executionId, "result", result.Status)
	if c.IsLegacy() {
		return c.legacyUpdateExecutionResult(ctx, environmentId, executionId, result)
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	req := &cloud.UpdateExecutionResultRequest{
		Id:     executionId,
		Result: resultBytes,
	}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.UpdateExecutionResult, req)
	return err
}

// Deprecated
func (c *client) legacyUpdateExecutionResult(ctx context.Context, environmentId, executionId string, result *testkube.TestWorkflowResult) error {
	jsonPayload, err := json.Marshal(cloudtestworkflow.ExecutionUpdateResultRequest{
		ID:     executionId,
		Result: result,
	})
	if err != nil {
		return err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowExecutionUpdateResult),
		Payload: &s,
	}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.Call, &req)
	return err
}

func (c *client) FinishExecutionResult(ctx context.Context, environmentId, executionId string, result *testkube.TestWorkflowResult) error {
	log.DefaultLogger.Debugw("grpc.FinishExecutionResult", "id", executionId)
	if c.IsLegacy() {
		return c.legacyUpdateExecutionResult(ctx, environmentId, executionId, result)
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	req := &cloud.FinishExecutionRequest{
		Id:     executionId,
		Result: resultBytes,
	}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.FinishExecution, req)
	return err
}

func (c *client) InitExecution(ctx context.Context, environmentId, executionId string, signature []testkube.TestWorkflowSignature, namespace string) error {
	log.DefaultLogger.Debugw("grpc.InitExecution", "id", executionId)
	if c.IsLegacy() {
		return c.legacyInitExecution(ctx, environmentId, executionId, signature, namespace)
	}

	signatureBytes, err := json.Marshal(signature)
	if err != nil {
		return err
	}
	req := &cloud.InitExecutionRequest{
		Id:        executionId,
		Namespace: namespace,
		Signature: signatureBytes,
	}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.InitExecution, req)
	return err
}

// Deprecated
func (c *client) legacyInitExecution(ctx context.Context, environmentId, executionId string, signature []testkube.TestWorkflowSignature, namespace string) error {
	execution, err := c.GetExecution(ctx, environmentId, executionId)
	if err != nil {
		return err
	}
	execution.RunnerId = c.proContext.Agent.ID
	execution.Namespace = namespace
	execution.Signature = signature
	execution.AssignedAt = time.Now()
	jsonPayload, err := json.Marshal(cloudtestworkflow.ExecutionUpdateRequest{
		WorkflowExecution: *execution,
	})
	if err != nil {
		return err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowExecutionUpdate),
		Payload: &s,
	}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.Call, &req)
	return err
}

// Deprecated
func (c *client) legacyUpdateExecutionOutput(ctx context.Context, environmentId, executionId string, output []testkube.TestWorkflowOutput) error {
	jsonPayload, err := json.Marshal(cloudtestworkflow.ExecutionUpdateOutputRequest{
		ID:     executionId,
		Output: output,
	})
	if err != nil {
		return err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowExecutionUpdateOutput),
		Payload: &s,
	}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.Call, &req)
	return err
}
