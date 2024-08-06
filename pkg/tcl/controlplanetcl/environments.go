package controlplanetcl

import (
	"context"
	"encoding/json"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/controlplane"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"google.golang.org/grpc"
)

func GetEnvironment(ctx context.Context, proContext config.ProContext, cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn) (resp GetEnvironmentResponse, err error) {
	executor := executor.NewCloudGRPCExecutor(cloudClient, grpcConn, proContext.APIKey, proContext.RunnerId)

	req := GetEnvironmentRequest{}
	respBytes, err := executor.Execute(ctx, controlplane.CmdControlPlaneGetEnvironment, req)
	if err != nil {
		return GetEnvironmentResponse{}, err
	}

	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return GetEnvironmentResponse{}, err
	}

	return resp, nil
}

// GetEnvironmentRequest represents a request to get an environment by the token
type GetEnvironmentRequest struct {
	Token string
}

// GetEnvironmentResponse represents a response with env org data of connected runner
type GetEnvironmentResponse struct {
	Id   string
	Name string
	Slug string

	OrganizationId   string
	OrganizationSlug string
	OrganizationName string
}
