package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

type ExecutionsReader channels.Watcher[testkube.TestWorkflowExecution]

// stickyChildRunningContextType returns the cloud RunningContextType that a
// child scheduled from a parent with the given actor type should carry, if the
// parent belongs to a "sticky actor" family. The bool is false for
// non-sticky parents; callers should fall back to the default
// RunningContextType_EXECUTION in that case.
//
// Only Quality Loop is sticky today. Extend this set carefully — every actor
// added here changes the wire semantics of every chained child scheduled by
// that actor's parents.
func stickyChildRunningContextType(parentActorType testkube.TestWorkflowRunningContextActorType) (cloud.RunningContextType, bool) {
	switch parentActorType {
	case testkube.QUALITYLOOP_TestWorkflowRunningContextActorType:
		return cloud.RunningContextType_QUALITYLOOP, true
	}
	return cloud.RunningContextType_UNKNOWN, false
}

type ExecutionSelfClient interface {
	AppendExecutionReport(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath string, report []byte) error
	SaveExecutionArtifactGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath, contentType string) (string, error)
	ScheduleExecution(ctx context.Context, environmentId string, request *cloud.ScheduleRequest) ExecutionsReader
	GetExecution(ctx context.Context, environmentId, executionId string) (*testkube.TestWorkflowExecution, error)
	GetCredential(ctx context.Context, environmentId, executionId, name string) ([]byte, error)
	GetCredentialWithSource(ctx context.Context, environmentId, executionId, name, source string) ([]byte, error)
	GetGitHubToken(ctx context.Context, url string) (string, error)
}

func (c *client) AppendExecutionReport(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath string, report []byte) error {
	req := cloud.AppendExecutionReportRequest{
		Id:       executionId,
		Step:     stepRef,
		FilePath: filePath,
		Report:   report,
	}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.AppendExecutionReport, &req)
	return err
}

func (c *client) SaveExecutionArtifactGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath, contentType string) (string, error) {
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

func (c *client) ScheduleExecution(ctx context.Context, environmentId string, request *cloud.ScheduleRequest) ExecutionsReader {
	if c.opts.ExecutionID != "" {
		// By default a chained child carries RunningContextType_EXECUTION so
		// the server maps it to actor.type=testworkflow. If the parent belongs
		// to a "sticky actor" family, propagate the parent's actor type to the
		// child instead: children then carry the parent's actor natively.
		requestType := cloud.RunningContextType_EXECUTION
		if stickyRunningContextType, ok := stickyChildRunningContextType(c.opts.ParentActorType); ok {
			requestType = stickyRunningContextType
		}
		request.RunningContext = &cloud.RunningContext{
			Name: c.opts.WorkflowName,
			Id:   c.opts.ExecutionID,
			Type: requestType,
		}
		request.ParentExecutionIds = append(c.opts.ParentExecutionIDs, c.opts.ExecutionID)
	}

	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ScheduleExecution, request)
	if err != nil {
		return channels.NewError[testkube.TestWorkflowExecution](err)
	}
	watcher := channels.NewWatcher[testkube.TestWorkflowExecution]()
	go func() {
		defer func() {
			watcher.Close(err)
		}()
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
	return c.GetCredentialWithSource(ctx, environmentId, executionId, name, "credential")
}

func (c *client) GetCredentialWithSource(ctx context.Context, environmentId, executionId, name, source string) ([]byte, error) {
	req := cloud.CredentialRequest{
		Name:        name,
		ExecutionId: executionId,
		Source:      source,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetCredential, &req)
	if err != nil {
		return nil, err
	}
	return res.Content, nil
}

func (c *client) GetGitHubToken(ctx context.Context, url string) (string, error) {
	req := cloud.GetGitHubTokenRequest{
		Url: url,
	}
	res, err := call(ctx, c.metadata().GRPC(), c.client.GetGitHubToken, &req)
	if err != nil {
		return "", err
	}
	return res.Token, nil
}
