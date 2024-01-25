package repository

import (
	"bufio"
	"context"
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
	ch := make(chan events.LogResponse, defaultBufferSize)
	file, err := r.storageClient.DownloadFileFromBucket(ctx, r.bucket, "", id)
	if err != nil {
		r.log.Errorw("error downloading log file from bucket", "error", err)
		return ch, err
	}

	reader := bufio.NewReader(file)
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
			return ch, err
		}

		// parse log line - also handle old (output.Output) and new format (just unstructured []byte)
		ch <- events.LogResponse{Log: events.NewLogResponseFromBytes(b)}
	}

	return ch, nil
}
