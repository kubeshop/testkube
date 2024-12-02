package credentials

import (
	"context"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"

	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
)

//go:generate mockgen -destination=./mock_repository.go -package=credentials "github.com/kubeshop/testkube/pkg/credentials" CredentialRepository
type CredentialRepository interface {
	Get(ctx context.Context, name string) ([]byte, error)
}

type credentialRepository struct {
	getClient   func() cloud.TestKubeCloudAPIClient
	apiKey      string
	executionId string
}

func NewCredentialRepository(getClient func() cloud.TestKubeCloudAPIClient, apiKey, executionId string) CredentialRepository {
	return &credentialRepository{getClient: getClient, apiKey: apiKey, executionId: executionId}
}

func (c *credentialRepository) Get(ctx context.Context, name string) ([]byte, error) {
	ctx = agentclient.AddAPIKeyMeta(ctx, c.apiKey)
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	result, err := c.getClient().GetCredential(ctx, &cloud.CredentialRequest{Name: name, ExecutionId: c.executionId}, opts...)
	if err != nil {
		return nil, err
	}
	return result.Content, nil
}
