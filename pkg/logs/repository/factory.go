package repository

import (
	"errors"

	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/state"
	"github.com/kubeshop/testkube/pkg/storage"
)

var ErrUnknownState = errors.New("unknown state")

type Factory interface {
	GetRepository(state state.LogState) (LogsRepository, error)
}

func NewJsMinioFactory(storageClient storage.ClientBucket, bucket string, logStream client.StreamGetter) Factory {
	return JsMinioFactory{
		storageClient: storageClient,
		bucket:        bucket,
		logStream:     logStream,
	}
}

type JsMinioFactory struct {
	storageClient storage.ClientBucket
	bucket        string
	logStream     client.StreamGetter
}

func (b JsMinioFactory) GetRepository(s state.LogState) (LogsRepository, error) {
	switch s {
	// pending get from buffer
	case state.LogStatePending:
		return NewJetstreamRepository(b.logStream), nil
	case state.LogStateFinished, state.LogStateUnknown:
		return NewMinioRepository(b.storageClient, b.bucket), nil
	default:
		return nil, ErrUnknownState
	}
}
