package repository

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/storage"
)

const (
	defaultBufferSize = 100
)

func NewMinioRepository(storageClient storage.ClientBucket, bucket string) LogsRepository {
	return MinioLogsRepository{
		storageClient: storageClient,
		log:           log.DefaultLogger,
		bucket:        bucket,
	}
}

type MinioLogsRepository struct {
	storageClient storage.ClientBucket
	log           *zap.SugaredLogger
	bucket        string
}

func (r MinioLogsRepository) Get(ctx context.Context, id string) (chan events.LogResponse, error) {
	ch := make(chan events.LogResponse, defaultBufferSize)
	file, err := r.storageClient.DownloadFileFromBucket(ctx, r.bucket, "", id)
	if err != nil {
		r.log.Errorw("error downloading log file from bucket", "error", err)
		return ch, err
	}

	var output []events.Log
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&output)
	if err != nil {
		r.log.Errorw("error decoding log lines", "error", err)
		return ch, err
	}

	for _, log := range output {
		ch <- events.LogResponse{Log: log}
	}

	return ch, nil
}
