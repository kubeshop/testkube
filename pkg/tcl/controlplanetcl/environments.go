package controlplanetcl

import (
	"context"

	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/controlplane"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

func GetEnvironment(ctx context.Context, proContext config.ProContext, cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn) (resp GetEnvironmentResponse, err error) {
	ex := executor.NewCloudGRPCExecutor(cloudClient, grpcConn, proContext.APIKey, proContext.RunnerId)
	bytes, err := ex.Execute(ctx, controlplane.CmdControlPlaneGetEnvironment, GetEnvironmentRequest{})
	if err != nil {
		return resp, err
	}
	return executor.ToResponse[GetEnvironmentResponse](bytes)
}

// GetEnvironmentRequest represents a request to get an environment by the token
type GetEnvironmentRequest struct {
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
