package repository

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	defaultBufferSize = 100
	logsV1Prefix      = "{\"id\""
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
	file, info, err := r.storageClient.DownloadFileFromBucket(ctx, r.bucket, "", id)
	if err != nil {
		r.log.Errorw("error downloading log file from bucket", "error", err, "bucket", r.bucket, "id", id)
		return nil, err
	}

	ch := make(chan events.LogResponse, defaultBufferSize)

	go func() {
		defer close(ch)

		buffer, version, err := r.readLineLogsV2(file, ch)
		if err != nil {
			ch <- events.LogResponse{Error: err}
			return
		}

		if version == events.LogVersionV1 {
			if err = r.readLineLogsV1(ch, buffer, info.LastModified); err != nil {
				ch <- events.LogResponse{Error: err}
				return
			}
		}
	}()

	return ch, nil
}

func (r MinioLogsRepository) readLineLogsV2(file io.Reader, ch chan events.LogResponse) ([]byte, events.LogVersion, error) {
	var buffer []byte
	reader := bufio.NewReader(file)
	firstLine := false
	version := events.LogVersionV2
	for {
		b, err := utils.ReadLongLine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			r.log.Errorw("error getting log line", "error", err)
			return nil, "", err
		}

		if !firstLine {
			firstLine = true
			if strings.HasPrefix(string(b), logsV1Prefix) {
				version = events.LogVersionV1
			}
		}

		if version == events.LogVersionV1 {
			buffer = append(buffer, b...)
		}

		if version == events.LogVersionV2 {
			var log events.Log
			err = json.Unmarshal(b, &log)
			if err != nil {
				r.log.Errorw("error unmarshalling log line", "error", err)
				ch <- events.LogResponse{Error: err}
				continue
			}

			ch <- events.LogResponse{Log: log}
		}
	}

	return buffer, version, nil
}

func (r MinioLogsRepository) readLineLogsV1(ch chan events.LogResponse, buffer []byte, logTime time.Time) error {
	var output result.ExecutionOutput
	decoder := json.NewDecoder(bytes.NewBuffer(buffer))
	err := decoder.Decode(&output)
	if err != nil {
		r.log.Errorw("error decoding logs", "error", err)
		return err
	}

	reader := bufio.NewReader(bytes.NewBuffer([]byte(output.Output)))
	for {
		b, err := utils.ReadLongLine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			r.log.Errorw("error getting log line", "error", err)
			return err
		}

		ch <- events.LogResponse{Log: events.Log{
			Time:    logTime,
			Content: string(b),
			Version: string(events.LogVersionV1),
		}}
	}

	return nil
}
