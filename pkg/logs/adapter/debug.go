package adapter

import (
	"context"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

var _ Adapter = &DebugAdapter{}

// NewDebugAdapter creates new DebugAdapter which will write logs to stdout
func NewDebugAdapter() *DebugAdapter {
	return &DebugAdapter{
		l: log.DefaultLogger,
	}
}

type DebugAdapter struct {
	l *zap.SugaredLogger
}

func (s *DebugAdapter) Init(ctx context.Context, id string) error {
	s.l.Debugw("Initializing", "id", id)
	return nil
}

func (s *DebugAdapter) Notify(ctx context.Context, id string, e events.Log) error {
	s.l.Debugw("got event", "id", id, "event", e)
	return nil
}

func (s *DebugAdapter) Stop(ctx context.Context, id string) error {
	s.l.Debugw("Stopping", "id", id)
	return nil
}

func (s *DebugAdapter) Name() string {
	return "dummy"
}
