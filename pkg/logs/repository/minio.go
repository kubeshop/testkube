package repository

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	defaultBufferSize = 100
)

func NewMinioRepository(minioClient *minio.Client, bucket string) LogsRepository {
	return MinioLogsRepository{
		minioClient: minioClient,
		log:         log.DefaultLogger,
		bucket:      bucket,
	}
}

type MinioLogsRepository struct {
	minioClient *minio.Client
	log         *zap.SugaredLogger
	bucket      string
}

func (r MinioLogsRepository) Get(ctx context.Context, id string) chan events.LogResponse {
	ch := make(chan events.LogResponse, defaultBufferSize)
	buffer := &bytes.Buffer{}
	exists, err := r.minioClient.BucketExists(ctx, r.bucket)
	if err != nil {
		ch <- events.LogResponse{Error: err}
		r.log.Errorw("error checking bucket", "err", err)
		return ch
	}

	if !exists {
		ch <- events.LogResponse{Error: fmt.Errorf("bucket doesn't exist %s", r.bucket)}
		r.log.Infow("bucket doesn't exist", "bucket", r.bucket)
		return ch
	}

	objInfo, err := r.minioClient.GetObject(ctx, r.bucket, id, minio.GetObjectOptions{})
	if err != nil {
		ch <- events.LogResponse{Error: err}
		r.log.Errorw("error getting object", "error", err)
		return ch
	}

	if _, err = objInfo.Stat(); err != nil {
		ch <- events.LogResponse{Error: err}
		r.log.Errorw("error getting object statistics", "error", err)
		return ch
	}

	if _, err = buffer.ReadFrom(objInfo); err != nil {
		ch <- events.LogResponse{Error: err}
		r.log.Errorw("error reading object", "err", err)
		return ch
	}

	r.log.Debugw("repository starts reading log lines")
	reader := bufio.NewReader(buffer)
	for {
		b, err := utils.ReadLongLine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			break
		}

		if err != nil {
			ch <- events.LogResponse{Error: err}
			r.log.Errorw("error getting log line", "error", err)
			return ch
		}

		// parse log line - also handle old (output.Output) and new format (just unstructured []byte)
		ch <- events.LogResponse{Log: events.NewLogResponseFromBytes(b)}
	}

	return ch
}
