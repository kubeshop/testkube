package credentials

import (
	"context"
	"time"

	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
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

//go:generate go tool mockgen -destination=./mock_repository.go -package=credentials "github.com/kubeshop/testkube/pkg/credentials" CredentialRepository
type CredentialRepository interface {
	Get(ctx context.Context, name string) ([]byte, error)
	GetWithSource(ctx context.Context, name, source string) ([]byte, error)
}

type credentialRepository struct {
	getClient     func() controlplaneclient.Client
	environmentId string
	executionId   string
}

func NewCredentialRepository(getClient func() controlplaneclient.Client, environmentId string, executionId string) CredentialRepository {
	return &credentialRepository{getClient: getClient, environmentId: environmentId, executionId: executionId}
}

func (c *credentialRepository) Get(ctx context.Context, name string) ([]byte, error) {
	return c.GetWithSource(ctx, name, SourceCredential)
}

func (c *credentialRepository) GetWithSource(ctx context.Context, name, source string) ([]byte, error) {
	var err error
	var result []byte
	for i := 0; i < GetCredentialRetryCount; i++ {
		result, err = c.getClient().GetCredentialWithSource(ctx, c.environmentId, c.executionId, name, source)
		if err == nil {
			return result, nil
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
