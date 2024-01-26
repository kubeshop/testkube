package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/utils"
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
	file, err := r.storageClient.DownloadFileFromBucket(ctx, r.bucket, "", id)
	if err != nil {
		r.log.Errorw("error downloading log file from bucket", "error", err)
		return nil, err
	}

	ch := make(chan events.LogResponse, defaultBufferSize)
	reader := bufio.NewReader(file)
	go func() {
		defer close(ch)

		for {
			b, err := utils.ReadLongLine(reader)
			if err != nil {
				if errors.Is(err, io.EOF) {
					err = nil
				}
				break
			}

			if err != nil {
				r.log.Errorw("error getting log line", "error", err)
				ch <- events.LogResponse{Error: err}
				return
			}

			var log events.Log
			err = json.Unmarshal(b, &log)
			if err != nil {
				r.log.Errorw("error unmarshalling log line", "error", err)
				ch <- events.LogResponse{Error: err}
				continue
			}

			ch <- events.LogResponse{Log: log}
		}
	}()

	return ch, nil
}
