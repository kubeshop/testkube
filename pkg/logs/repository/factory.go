package repository

import (
	"errors"

	"github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

var ErrUnknownState = errors.New("unknown state")

type Factory interface {
	GetRepository(state state.LogState) (LogsRepository, error)
}

func NewJsMinioFactory(minio *minio.Client, bucket string, logStream client.Stream) Factory {
	return JsMinioFactory{
		minio:  minio,
		bucket: bucket,
		js:     js,
	}
}

type JsMinioFactory struct {
	minio     *minio.Client
	bucket    string
	logStream client.Stream
}

func (b JsMinioFactory) GetRepository(s state.LogState) (LogsRepository, error) {
	switch s {
	// pending get from buffer
	case state.LogStatePending:
		return NewJetstreamRepository(b.js), nil
	case state.LogStateFinished:
		return NewMinioRepository(b.minio, b.bucket), nil
	default:
		return nil, ErrUnknownState
	}
}
