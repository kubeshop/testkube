package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	signaturev1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/signature/v1"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func (s *Server) SetExecutionScheduling(ctx context.Context, req *executionv1.SetExecutionSchedulingRequest) (*executionv1.SetExecutionSchedulingResponse, error) {
	execution, err := s.resultsRepository.Get(ctx, req.GetExecutionId())
	if err != nil {
		return nil, errors.Join(
			status.Error(codes.NotFound, "execution does not exist"),
			fmt.Errorf("retrieve execution to set scheduling: %w", err),
		)
	}
	if err := s.resultsRepository.Init(ctx, req.GetExecutionId(), testworkflow.InitData{
		RunnerID:   execution.RunnerId,
		Namespace:  req.GetNamespace(),
		Signature:  translateSignature(req.GetSignature()),
		AssignedAt: execution.AssignedAt,
	}); err != nil {
		return nil, errors.Join(
			status.Error(codes.Internal, "failed to set execution scheduling"),
			fmt.Errorf("set execution scheduling: %w", err),
		)
	}

	return &executionv1.SetExecutionSchedulingResponse{}, nil
}

func translateSignature(sigs []*signaturev1.Signature) []testkube.TestWorkflowSignature {
	var ret []testkube.TestWorkflowSignature
	for _, sig := range sigs {
		ret = append(ret, testkube.TestWorkflowSignature{
			Ref:      sig.GetRef(),
			Name:     sig.GetName(),
			Category: sig.GetCategory(),
			Optional: sig.GetOptional(),
			Negative: sig.GetNegative(),
			Children: translateSignature(sig.GetChildren()),
		})
	}
	return ret
}

func (s *Server) Register(ctx context.Context, request *cloud.RegisterRequest) (*cloud.RegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not supported in the standalone version")
}

func (s *Server) GetUnfinishedExecutions(_ *emptypb.Empty, srv cloud.TestKubeCloudAPI_GetUnfinishedExecutionsServer) error {
	executions, err := s.resultsRepository.GetExecutions(srv.Context(), testworkflow.FilterImpl{
		FStatuses: testkube.TestWorkflowExecutingStatus,
		FPageSize: math.MaxInt32,
	})
	if err != nil {
		return err
	}
	for _, execution := range executions {
		err = srv.Send(&cloud.UnfinishedExecution{EnvironmentId: common.StandaloneEnvironment, Id: execution.Id})
		if err != nil {
			return err
		}
	}
	return nil
}

// Deprecated: superseded by testkube.testworkflow.execution.v1/TestWorkflowExecutionService.GetExecutionUpdates
func (s *Server) GetRunnerRequests(srv cloud.TestKubeCloudAPI_GetRunnerRequestsServer) error {
	// Do nothing - it doesn't need to send runner requests
	<-srv.Context().Done()
	return nil
}

// Deprecated: superseded by testkube.testworkflow.execution.v1/TestWorkflowExecutionService.SetExecutionScheduling
func (s *Server) InitExecution(ctx context.Context, req *cloud.InitExecutionRequest) (*cloud.InitExecutionResponse, error) {
	var signature []testkube.TestWorkflowSignature
	err := json.Unmarshal(req.Signature, &signature)
	if err != nil {
		return nil, err
	}
	err = s.resultsRepository.Init(ctx, req.Id, testworkflow.InitData{RunnerID: "oss", Namespace: req.Namespace, Signature: signature, AssignedAt: time.Now()})
	if err != nil {
		return nil, err
	}
	return &cloud.InitExecutionResponse{}, nil
}

func (s *Server) GetExecution(ctx context.Context, req *cloud.GetExecutionRequest) (*cloud.GetExecutionResponse, error) {
	execution, err := s.resultsRepository.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	executionBytes, err := json.Marshal(execution)
	if err != nil {
		return nil, err
	}
	return &cloud.GetExecutionResponse{Execution: executionBytes}, nil
}

func (s *Server) UpdateExecutionResult(ctx context.Context, req *cloud.UpdateExecutionResultRequest) (*cloud.UpdateExecutionResultResponse, error) {
	var result testkube.TestWorkflowResult
	err := json.Unmarshal(req.Result, &result)
	if err != nil {
		return nil, err
	}
	err = s.resultsRepository.UpdateResult(ctx, req.Id, &result)
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateExecutionResultResponse{}, nil
}

func (s *Server) UpdateExecutionOutput(ctx context.Context, req *cloud.UpdateExecutionOutputRequest) (*cloud.UpdateExecutionOutputResponse, error) {
	err := s.resultsRepository.UpdateOutput(ctx, req.Id, common.MapSlice(req.Output, func(t *cloud.ExecutionOutput) testkube.TestWorkflowOutput {
		var v map[string]interface{}
		_ = json.Unmarshal(t.Value, &v)
		return testkube.TestWorkflowOutput{Ref: t.Ref, Name: t.Name, Value: v}
	}))
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateExecutionOutputResponse{}, nil
}

func (s *Server) SaveExecutionLogsPresigned(ctx context.Context, req *cloud.SaveExecutionLogsPresignedRequest) (*cloud.SaveExecutionLogsPresignedResponse, error) {
	url, err := s.outputRepository.PresignSaveLog(ctx, req.Id, "")
	if err != nil {
		return nil, err
	}
	return &cloud.SaveExecutionLogsPresignedResponse{Url: url}, nil
}

func (s *Server) FinishExecution(ctx context.Context, req *cloud.FinishExecutionRequest) (*cloud.FinishExecutionResponse, error) {
	var result testkube.TestWorkflowResult
	err := json.Unmarshal(req.Result, &result)
	if err != nil {
		return nil, err
	}
	err = s.resultsRepository.UpdateResult(ctx, req.Id, &result)
	if err != nil {
		return nil, err
	}
	return &cloud.FinishExecutionResponse{}, nil
}

func (s *Server) GetGitHubToken(_ context.Context, _ *cloud.GetGitHubTokenRequest) (*cloud.GetGitHubTokenResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "github integration is not supported")
}
