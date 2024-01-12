package repository

import (
	"context"

	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

var _ LogsRepository = &JetstreamLogsRepository{}

func NewJetstreamRepository(client client.StreamGetter) LogsRepository {
	return JetstreamLogsRepository{c: client}
}

// Jet
type JetstreamLogsRepository struct {
	c client.StreamGetter
}

func (r JetstreamLogsRepository) Get(ctx context.Context, id string) (chan events.LogResponse, error) {
	return r.c.Get(ctx, id)
}
