package repository

import (
	"errors"

	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/state"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

var ErrUnknownState = errors.New("unknown state")

type Factory interface {
	GetRepository(state state.LogState) (LogsRepository, error)
}

type JsMinioFactory struct {
	minio *minio.Client
	js    client.StreamGetter
}

func (b JsMinioFactory) GetRepository(s state.LogState) (LogsRepository, error) {
	switch s {
	// pending get from buffer
	case state.LogStatePending:
		return NewJetstreamRepository(b.js), nil
	case state.LogStateFinished:
		return NewMinioRepository(b.minio), nil
	default:
		return nil, ErrUnknownState
	}
}
