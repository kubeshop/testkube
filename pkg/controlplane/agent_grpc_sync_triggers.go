package controlplane

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func (s *Server) GetTestTrigger(_ context.Context, _ *cloud.GetTestTriggerRequest) (*cloud.GetTestTriggerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) ListTestTriggers(_ *cloud.ListTestTriggersRequest, _ cloud.TestKubeCloudAPI_ListTestTriggersServer) error {
	return status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) ListTestTriggerLabels(_ context.Context, _ *cloud.ListTestTriggerLabelsRequest) (*cloud.ListTestTriggerLabelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) CreateTestTrigger(_ context.Context, _ *cloud.CreateTestTriggerRequest) (*cloud.CreateTestTriggerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) UpdateTestTrigger(_ context.Context, _ *cloud.UpdateTestTriggerRequest) (*cloud.UpdateTestTriggerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) DeleteTestTrigger(_ context.Context, _ *cloud.DeleteTestTriggerRequest) (*cloud.DeleteTestTriggerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) DeleteAllTestTriggers(_ context.Context, _ *cloud.DeleteAllTestTriggersRequest) (*cloud.DeleteAllTestTriggersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) DeleteTestTriggersByLabels(_ context.Context, _ *cloud.DeleteTestTriggersByLabelsRequest) (*cloud.DeleteTestTriggersByLabelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) WatchTestTriggerUpdates(_ *cloud.WatchTestTriggerUpdatesRequest, _ cloud.TestKubeCloudAPI_WatchTestTriggerUpdatesServer) error {
	return status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}
