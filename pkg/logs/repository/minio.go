package repository

import (
	"context"

	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func NewMinioRepository(minio *minio.Client) LogsRepository {
	return MinioLogsRepository{}
}

type MinioLogsRepository struct {
}

func (r MinioLogsRepository) Get(ctx context.Context, id string) (chan events.LogResponse, error) {
	ch := make(chan events.LogResponse, 100)
	return ch, nil
}
