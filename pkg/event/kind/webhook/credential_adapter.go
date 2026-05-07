package webhook

import (
	"context"
	"fmt"

	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/credentials"
)

// grpcCredentialAdapter adapts cloud.TestKubeCloudAPIClient to credentials.CredentialRepository
// This is a lightweight adapter for webhook use where we don't have full controlplaneclient.Client
type grpcCredentialAdapter struct {
	grpcClient     cloud.TestKubeCloudAPIClient
	environmentId  string
	executionId    string
	apiKey         string
	organizationId string
	agentId        string
}

// newGRPCCredentialAdapter creates a new credential repository adapter for webhooks
func newGRPCCredentialAdapter(grpcClient cloud.TestKubeCloudAPIClient, environmentId string, executionId string, apiKey string, organizationId string, agentId string) credentials.CredentialRepository {
	return &grpcCredentialAdapter{
		grpcClient:     grpcClient,
		environmentId:  environmentId,
		executionId:    executionId,
		apiKey:         apiKey,
		organizationId: organizationId,
		agentId:        agentId,
	}
}

func (a *grpcCredentialAdapter) Get(ctx context.Context, name string) ([]byte, error) {
	return a.GetWithSource(ctx, name, credentials.SourceCredential)
}

func (a *grpcCredentialAdapter) GetWithSource(ctx context.Context, name, source string) ([]byte, error) {
	// Add authentication metadata to context using shared helper
	ctx = agentclient.AddMetadata(ctx, a.apiKey, a.organizationId, a.environmentId, a.agentId)

	req := &cloud.CredentialRequest{
		Name:        name,
		ExecutionId: a.executionId,
		Source:      source,
	}

	resp, err := a.grpcClient.GetCredential(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential %q (source: %s) from control plane: %w", name, source, err)
	}

	return resp.Content, nil
}
