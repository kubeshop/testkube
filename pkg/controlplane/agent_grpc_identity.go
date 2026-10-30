package controlplane

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/identitytoken"
)

// apiKeyMetadata carries the execution token, as it carries the agent key for
// the commercial control plane.
const apiKeyMetadata = "api-key"

// IdentityTokenSigner is implemented by *identitytoken.Signer.
type IdentityTokenSigner interface {
	Sign(subject identitytoken.Subject, audience string) (string, error)
	ExecutionToken(environmentID, executionID string) string
	VerifyExecutionToken(environmentID, executionID, token string) bool
}

// executionToken is the token handed to an execution's pods, or "" when
// identity tokens are off.
func (s *Server) executionToken(environmentID, executionID string) string {
	if s.cfg.IdentityTokens == nil {
		return ""
	}
	return s.cfg.IdentityTokens.ExecutionToken(environmentID, executionID)
}

func (s *Server) GetIdentityToken(ctx context.Context, req *cloud.GetIdentityTokenRequest) (*cloud.GetIdentityTokenResponse, error) {
	signer := s.cfg.IdentityTokens
	if signer == nil {
		return nil, status.Error(codes.FailedPrecondition, "identity tokens are not configured in this Control Plane")
	}
	if req.ExecutionId == "" {
		return nil, status.Error(codes.InvalidArgument, "execution ID is required")
	}
	// Nothing else on this gRPC port authenticates the caller, so without the
	// execution token any pod that reaches it could mint a token for any running
	// execution and act as its workflow.
	var token string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(apiKeyMetadata); len(values) > 0 {
			token = values[0]
		}
	}
	if !signer.VerifyExecutionToken(common.StandaloneEnvironment, req.ExecutionId, token) {
		return nil, status.Error(codes.PermissionDenied, "identity token not accessible")
	}

	execution, err := s.resultsRepository.Get(ctx, req.ExecutionId)
	if err != nil {
		return nil, err
	}
	// Get also matches by name, and the execution token only covers the requested
	// string, so a name match could mint a token for another workflow.
	if execution.Id != req.ExecutionId {
		return nil, status.Error(codes.NotFound, "execution not found")
	}
	if execution.Result != nil && execution.Result.Status != nil && execution.Result.IsFinished() {
		return nil, status.Error(codes.NotFound, "execution is expired")
	}
	if execution.Workflow == nil || execution.Workflow.Name == "" {
		return nil, status.Error(codes.FailedPrecondition, "execution has no workflow")
	}

	signed, err := signer.Sign(identitytoken.Subject{
		OrganizationID: s.cfg.OrganizationID,
		EnvironmentID:  s.envID,
		WorkflowName:   execution.Workflow.Name,
		ExecutionID:    req.ExecutionId,
		RunnerID:       execution.RunnerId,
	}, req.Audience)
	if errors.Is(err, identitytoken.ErrInvalidAudience) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "signing identity token: %v", err)
	}
	return &cloud.GetIdentityTokenResponse{Token: signed}, nil
}
