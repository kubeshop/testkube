package credentials

import (
	"context"
	"math"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"

	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/log"
)

type grpcstatus interface {
	GRPCStatus() *status.Status
}

const (
	GetCredentialRetryCount = 30
)

func getIterationDelay(iteration int) time.Duration {
	if iteration < 5 {
		return 500 * time.Millisecond
	} else if iteration < 100 {
		return 1 * time.Second
	}
	return 5 * time.Second
}

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
	var err error
	var result *cloud.CredentialResponse
	for i := 0; i < GetCredentialRetryCount; i++ {
		result, err = c.getClient().GetCredential(ctx, &cloud.CredentialRequest{Name: name, ExecutionId: c.executionId}, opts...)
		if err == nil {
			return result.Content, nil
		}
		if _, ok := err.(grpcstatus); ok {
			return nil, err
		}
		// Try to get credentials again if it may be recoverable error
		log.DefaultLogger.Warnw("failed to get credential", "error", err)
		time.Sleep(getIterationDelay(i))
	}
	return nil, err
}
