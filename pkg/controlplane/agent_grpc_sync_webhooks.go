package controlplane

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func (s *Server) GetWebhook(_ context.Context, _ *cloud.GetWebhookRequest) (*cloud.GetWebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) ListWebhooks(_ *cloud.ListWebhooksRequest, _ cloud.TestKubeCloudAPI_ListWebhooksServer) error {
	return status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) ListWebhookLabels(_ context.Context, _ *cloud.ListWebhookLabelsRequest) (*cloud.ListWebhookLabelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) CreateWebhook(_ context.Context, _ *cloud.CreateWebhookRequest) (*cloud.CreateWebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")

}

func (s *Server) UpdateWebhook(_ context.Context, _ *cloud.UpdateWebhookRequest) (*cloud.UpdateWebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) DeleteWebhook(_ context.Context, _ *cloud.DeleteWebhookRequest) (*cloud.DeleteWebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) DeleteAllWebhooks(_ context.Context, _ *cloud.DeleteAllWebhooksRequest) (*cloud.DeleteAllWebhooksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) DeleteWebhooksByLabels(_ context.Context, _ *cloud.DeleteWebhooksByLabelsRequest) (*cloud.DeleteWebhooksByLabelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}

func (s *Server) WatchWebhookUpdates(_ *cloud.WatchWebhookUpdatesRequest, _ cloud.TestKubeCloudAPI_WatchWebhookUpdatesServer) error {
	return status.Errorf(codes.Unimplemented, "gitops functionality is not supported")
}
