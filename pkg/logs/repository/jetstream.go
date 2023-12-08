package repository

import (
	"context"

	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

const (
	buffer = 100
)

var _ LogsRepository = &JetstreamLogsRepository{}

func NewJetstreamRepository(client client.Client) LogsRepository {
	return JetstreamLogsRepository{c: client}
}

// Jet
type JetstreamLogsRepository struct {
	c client.Client
}

func (r JetstreamLogsRepository) Get(ctx context.Context, id string) chan events.LogResponse {
	return r.c.Get(ctx, id)
}
