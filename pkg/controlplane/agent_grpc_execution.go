package controlplane

import (
	"context"
	"errors"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

// ArtifactPresignedURLExpiration is how long a granted artifact URL stays usable.
const ArtifactPresignedURLExpiration = 15 * time.Minute

func (s *Server) SaveExecutionArtifactPresigned(ctx context.Context, req *cloud.SaveExecutionArtifactPresignedRequest) (*cloud.SaveExecutionArtifactPresignedResponse, error) {
	url, err := s.storageClient.PresignUploadFileToBucket(ctx, s.cfg.StorageBucket, req.Id, req.FilePath, ArtifactPresignedURLExpiration)
	if err != nil {
		return nil, err
	}
	return &cloud.SaveExecutionArtifactPresignedResponse{Url: url}, nil
}

// ListExecutionArtifactsPresigned grants read access to the artifacts of an execution,
// so a workflow can consume what the test workflows it ran produced.
func (s *Server) ListExecutionArtifactsPresigned(ctx context.Context, req *cloud.ListExecutionArtifactsPresignedRequest) (*cloud.ListExecutionArtifactsPresignedResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "execution id is required")
	}

	files, err := s.storageClient.ListFilesFromBucket(ctx, s.cfg.StorageBucket, req.Id)
	if err != nil {
		if errors.Is(err, minio.ErrArtifactsNotFound) {
			return &cloud.ListExecutionArtifactsPresignedResponse{}, nil
		}
		return nil, err
	}

	artifacts := make([]*cloud.ExecutionArtifactRef, 0, len(files))
	for _, file := range files {
		if !matchesArtifactPatterns(file.Name, req.Patterns) {
			continue
		}
		url, err := s.storageClient.PresignDownloadFileFromBucket(ctx, s.cfg.StorageBucket, req.Id, file.Name, ArtifactPresignedURLExpiration)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, &cloud.ExecutionArtifactRef{Path: file.Name, Url: url, Size: int64(file.Size)})
	}
	return &cloud.ListExecutionArtifactsPresignedResponse{Artifacts: artifacts}, nil
}

// matchesArtifactPatterns reports whether the artifact is requested. No pattern means
// every artifact is.
func matchesArtifactPatterns(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if matched, err := doublestar.PathMatch(pattern, path); err == nil && matched {
			return true
		}
	}
	return false
}

func (s *Server) AppendExecutionReport(_ context.Context, _ *cloud.AppendExecutionReportRequest) (*cloud.AppendExecutionReportResponse, error) {
	// This is currently only used for CapabilityJUnitReports which is unsupported by OSS.
	return nil, status.Error(codes.Unimplemented, "not supported in the standalone version")
}
