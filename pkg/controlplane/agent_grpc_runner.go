package controlplane

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

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
		err = srv.Send(&cloud.UnfinishedExecution{Id: execution.Id})
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
