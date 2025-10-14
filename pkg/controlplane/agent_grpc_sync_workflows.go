package controlplane

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func (s *Server) GetTestWorkflow(ctx context.Context, req *cloud.GetTestWorkflowRequest) (*cloud.GetTestWorkflowResponse, error) {
	workflow, err := s.testWorkflowsClient.Get(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return nil, err
	}
	return &cloud.GetTestWorkflowResponse{Workflow: workflowBytes}, nil
}

func (s *Server) CreateTestWorkflow(ctx context.Context, req *cloud.CreateTestWorkflowRequest) (*cloud.CreateTestWorkflowResponse, error) {
	var workflow testkube.TestWorkflow
	err := json.Unmarshal(req.Workflow, &workflow)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowsClient.Create(ctx, "", workflow)
	if err != nil {
		return nil, err
	}
	return &cloud.CreateTestWorkflowResponse{}, nil
}

func (s *Server) UpdateTestWorkflow(ctx context.Context, req *cloud.UpdateTestWorkflowRequest) (*cloud.UpdateTestWorkflowResponse, error) {
	var workflow testkube.TestWorkflow
	err := json.Unmarshal(req.Workflow, &workflow)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowsClient.Update(ctx, "", workflow)
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateTestWorkflowResponse{}, nil
}

func (s *Server) DeleteTestWorkflow(ctx context.Context, req *cloud.DeleteTestWorkflowRequest) (*cloud.DeleteTestWorkflowResponse, error) {
	err := s.testWorkflowsClient.Delete(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowResponse{}, nil
}

func (s *Server) DeleteTestWorkflowsByLabels(ctx context.Context, req *cloud.DeleteTestWorkflowsByLabelsRequest) (*cloud.DeleteTestWorkflowsByLabelsResponse, error) {
	count, err := s.testWorkflowsClient.DeleteByLabels(ctx, "", req.Labels)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowsByLabelsResponse{Count: count}, nil
}

func (s *Server) ListTestWorkflows(*cloud.ListTestWorkflowsRequest, cloud.TestKubeCloudAPI_ListTestWorkflowsServer) error {
	return status.Errorf(codes.Unimplemented, "method ListTestWorkflows not implemented")
}
func (s *Server) ListTestWorkflowLabels(context.Context, *cloud.ListTestWorkflowLabelsRequest) (*cloud.ListTestWorkflowLabelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTestWorkflowLabels not implemented")
}
func (s *Server) WatchTestWorkflowUpdates(*cloud.WatchTestWorkflowUpdatesRequest, cloud.TestKubeCloudAPI_WatchTestWorkflowUpdatesServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchTestWorkflowUpdates not implemented")
}
