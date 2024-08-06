package controlplanetcl

import (
	"context"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/controlplane"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor/v2"
	"google.golang.org/grpc"
)

func GetEnvironment(ctx context.Context, proContext config.ProContext, cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn) (resp GetEnvironmentResponse, err error) {
	ex := executor.NewCloudGRPCExecutor[GetEnvironmentResponse](cloudClient, grpcConn, proContext.APIKey, proContext.RunnerId)

	return ex.Execute(ctx, executor.ExecuteParams{
		Command: controlplane.CmdControlPlaneGetEnvironment,
		Payload: GetEnvironmentRequest{},
	})
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
