package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

type ExecutionsReader channels.Watcher[testkube.TestWorkflowExecution]

type ExecutionSelfClient interface {
	AppendExecutionReport(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath string, report []byte) error
	SaveExecutionArtifactGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath, contentType string) (string, error)
	ScheduleExecution(ctx context.Context, environmentId string, request *cloud.ScheduleRequest) ExecutionsReader
	GetExecution(ctx context.Context, environmentId, executionId string) (*testkube.TestWorkflowExecution, error)
	GetCredential(ctx context.Context, environmentId, executionId, name string) ([]byte, error)
}

func (c *client) AppendExecutionReport(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath string, report []byte) error {
	if c.IsLegacy() {
		return c.legacyAppendExecutionReport(ctx, environmentId, executionId, legacyWorkflowName, stepRef, filePath, report)
	}
	req := cloud.AppendExecutionReportRequest{
		Id:       executionId,
		Step:     stepRef,
		FilePath: filePath,
		Report:   report,
	}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.AppendExecutionReport, &req)
	return err
}

// Deprecated
func (c *client) legacyAppendExecutionReport(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath string, report []byte) error {
	jsonPayload, err := json.Marshal(cloudtestworkflow.ExecutionsAddReportRequest{
		ID:           executionId,
		WorkflowName: legacyWorkflowName,
		WorkflowStep: stepRef,
		Filepath:     filePath,
		Report:       report,
	})
	if err != nil {
		return err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowExecutionAddReport),
		Payload: &s,
	}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.Call, &req)
	return err
}

func (c *client) SaveExecutionArtifactGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath, contentType string) (string, error) {
	if c.IsLegacy() {
		return c.legacySaveExecutionArtifactGetPresignedURL(ctx, environmentId, executionId, legacyWorkflowName, stepRef, filePath, contentType)
	}
	req := cloud.SaveExecutionArtifactPresignedRequest{
		Id:          executionId,
		Step:        stepRef,
		FilePath:    filePath,
		ContentType: contentType,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.SaveExecutionArtifactPresigned, &req)
	if err != nil {
		return "", err
	}
	return res.Url, nil
}

// Deprecated
func (c *client) legacySaveExecutionArtifactGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath, contentType string) (string, error) {
	jsonPayload, err := json.Marshal(artifact.PutObjectSignedURLRequest{
		ExecutionID:      executionId,
		TestWorkflowName: legacyWorkflowName,
		Object:           filePath,
		ContentType:      contentType,
	})
	if err != nil {
		return "", err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return "", err
	}
	req := cloud.CommandRequest{
		Command: string(artifact.CmdScraperPutObjectSignedURL),
		Payload: &s,
	}
	cmdResponse, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.Call, &req)
	if err != nil {
		return "", err
	}
	var response artifact.PutObjectSignedURLResponse
	err = json.Unmarshal(cmdResponse.Response, &response)
	return response.URL, err
}

func (c *client) ScheduleExecution(ctx context.Context, environmentId string, request *cloud.ScheduleRequest) ExecutionsReader {
	if c.IsLegacy() {
		return channels.NewError[testkube.TestWorkflowExecution](ErrNotSupported)
	}
	if c.opts.ExecutionID != "" {
		request.RunningContext = &cloud.RunningContext{
			Name: c.opts.ExecutionID,
			Type: cloud.RunningContextType_EXECUTION,
		}
		request.ParentExecutionIds = append(c.opts.ParentExecutionIDs, c.opts.ExecutionID)
	}

	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ScheduleExecution, request)
	if err != nil {
		return channels.NewError[testkube.TestWorkflowExecution](err)
	}
	watcher := channels.NewWatcher[testkube.TestWorkflowExecution]()
	go func() {
		defer watcher.Close(err)
		for {
			item, itemErr := res.Recv()
			if itemErr != nil {
				if !errors.Is(itemErr, io.EOF) {
					err = itemErr
				}
				return
			}
			var execution testkube.TestWorkflowExecution
			itemErr = json.Unmarshal(item.Execution, &execution)
			if itemErr != nil {
				err = itemErr
				return
			}
			watcher.Send(execution)
		}
	}()
	return watcher
}

func (c *client) GetCredential(ctx context.Context, environmentId, executionId, name string) ([]byte, error) {
	req := cloud.CredentialRequest{
		Name:        name,
		ExecutionId: executionId,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetCredential, &req)
	if err != nil {
		return nil, err
	}
	return res.Content, nil
}
