package credentials

import (
	"context"

	"github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
)

//go:generate mockgen -destination=./mock_repository.go -package=credentials "github.com/kubeshop/testkube/pkg/credentials" CredentialRepository
type CredentialRepository interface {
	Get(ctx context.Context, name string) ([]byte, error)
}

type credentialRepository struct {
	client cloud.TestKubeCloudAPIClient
	apiKey string
}

func NewCredentialRepository(client cloud.TestKubeCloudAPIClient, apiKey string) CredentialRepository {
	return &credentialRepository{client: client, apiKey: apiKey}
}

func (c *credentialRepository) Get(ctx context.Context, name string) ([]byte, error) {
	ctx = client.AddAPIKeyMeta(context.Background(), c.apiKey)
	result, err := c.client.GetCredential(ctx, &cloud.CredentialRequest{Name: name})
	if err != nil {
		return nil, err
	}
	return result.Content, nil
}
