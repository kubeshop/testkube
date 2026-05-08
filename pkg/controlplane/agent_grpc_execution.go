package controlplane

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func (s *Server) SaveExecutionArtifactPresigned(ctx context.Context, req *cloud.SaveExecutionArtifactPresignedRequest) (*cloud.SaveExecutionArtifactPresignedResponse, error) {
	url, err := s.storageClient.PresignUploadFileToBucket(ctx, s.cfg.StorageBucket, req.Id, req.FilePath, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	return &cloud.SaveExecutionArtifactPresignedResponse{Url: url}, nil
}

func (s *Server) AppendExecutionReport(_ context.Context, _ *cloud.AppendExecutionReportRequest) (*cloud.AppendExecutionReportResponse, error) {
	// This is currently only used for CapabilityJUnitReports which is unsupported by OSS.
	return nil, status.Error(codes.Unimplemented, "not supported in the standalone version")
}
