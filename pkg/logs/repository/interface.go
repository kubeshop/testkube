package repository

import (
	"context"

	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

// RepositoryBuilder is responsible for getting valid repository based on execution state
// It'll be ususally for OSS when we'll get from NATS buffer or from Minio (when execution completed)
type RepositoryBuilder interface {
	GetRepository(state state.LogState) (LogsRepository, error)
}

// LogsRepository is the repository primitive to get logs from
type LogsRepository interface {
	Get(ctx context.Context, id string) (chan events.LogResponse, error)
}
