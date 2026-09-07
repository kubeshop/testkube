package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
	"github.com/kubeshop/testkube/pkg/testresults"
)

type ExecutionsReader channels.Watcher[testkube.TestWorkflowExecution]

type ExecutionSelfClient interface {
	AppendExecutionReport(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath string, report []byte, digest *testresults.Digest) error
	SaveExecutionArtifactGetPresignedURL(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath, contentType string) (string, error)
	ListExecutionArtifactsGetPresignedURLs(ctx context.Context, environmentId, executionId string, patterns []string) ([]ExecutionArtifact, error)
	ScheduleExecution(ctx context.Context, environmentId string, request *cloud.ScheduleRequest) ExecutionsReader
	GetExecution(ctx context.Context, environmentId, executionId string) (*testkube.TestWorkflowExecution, error)
	GetCredential(ctx context.Context, environmentId, executionId, name string) ([]byte, error)
	GetCredentialWithSource(ctx context.Context, environmentId, executionId, name, source string) ([]byte, error)
	GetGitHubToken(ctx context.Context, url string) (string, error)
}

func (c *client) AppendExecutionReport(ctx context.Context, environmentId, executionId, legacyWorkflowName, stepRef, filePath string, report []byte, digest *testresults.Digest) error {
	req := cloud.AppendExecutionReportRequest{
		Id:       executionId,
		Step:     stepRef,
		FilePath: filePath,
		// Still sent for a control plane that predates the parsed fields. One
		// that reads Summary should ignore it.
		//
		//nolint:staticcheck // the generated marker is stale: the proto comment
		// said "Deprecated:" before it was reworded, and the field is required
		// rather than discouraged. Drop this once the protobuf is regenerated.
		Report: report,
	}
	applyReportDigest(&req, digest)
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.AppendExecutionReport, &req)
	return err
}

// applyReportDigest fills the parsed fields of the request.
//
// Muted, Unexpected and Tolerated are deliberately left unset: the report says
// which test cases failed, while whether a failure was tolerated is the step's
// verdict, decided in a different container from the one uploading artifacts.
// The step result carries that, and the two are joined by step reference.
func applyReportDigest(req *cloud.AppendExecutionReportRequest, digest *testresults.Digest) {
	if digest == nil {
		return
	}

	req.Kind = "junit"
	req.Summary = &cloud.TestReportSummary{
		Tests:    digest.Counts.Tests,
		Passed:   digest.Counts.Passed,
		Failed:   digest.Counts.Failed,
		Skipped:  digest.Counts.Skipped,
		Errored:  digest.Counts.Errored,
		Duration: digest.Counts.DurationMs,
	}
	req.FailuresTruncated = digest.Truncated
	req.Failures = make([]*cloud.TestCaseFailure, 0, len(digest.Failures))
	for _, failure := range digest.Failures {
		req.Failures = append(req.Failures, &cloud.TestCaseFailure{
			Id:      failure.Id,
			Status:  string(failure.Status),
			Message: failure.Message,
		})
	}
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

// ExecutionArtifact is a single artifact of an execution, along with a temporary URL
// to download it from.
type ExecutionArtifact struct {
	Path string
	Url  string
	Size int64
}

func (c *client) ListExecutionArtifactsGetPresignedURLs(ctx context.Context, environmentId, executionId string, patterns []string) ([]ExecutionArtifact, error) {
	req := cloud.ListExecutionArtifactsPresignedRequest{
		Id:       executionId,
		Patterns: patterns,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListExecutionArtifactsPresigned, &req)
	if err != nil {
		return nil, err
	}
	artifacts := make([]ExecutionArtifact, 0, len(res.Artifacts))
	for _, artifact := range res.Artifacts {
		artifacts = append(artifacts, ExecutionArtifact{Path: artifact.Path, Url: artifact.Url, Size: artifact.Size})
	}
	return artifacts, nil
}

func (c *client) ScheduleExecution(ctx context.Context, environmentId string, request *cloud.ScheduleRequest) ExecutionsReader {
	if c.opts.ExecutionID != "" {
		request.RunningContext = &cloud.RunningContext{
			Name: c.opts.ParentActorType.ChildRunningContextName(c.opts.WorkflowName),
			Id:   c.opts.ExecutionID,
			Type: c.opts.ParentActorType.ChildRunningContextType(),
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
