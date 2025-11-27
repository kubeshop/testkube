package triggers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/coordination/leader"
	"github.com/kubeshop/testkube/pkg/event/bus"
)

func TestService_NewServiceReturnsTasks(t *testing.T) {
	logger := zap.NewNop().Sugar()
	externalCoordinator := leader.New(leaseBackendNoop{}, "external-id", "cluster", logger, leader.WithCheckInterval(time.Hour))

	tasks := NewService(
		"agent",
		nil,
		nil,
		nil,
		nil,
		logger,
		bus.NewEventBusMock(),
		metrics.NewMetrics(),
		nil,
		nil,
		nil,
		nil,
		WithCoordinator(externalCoordinator),
	)

	assert.Len(t, tasks, 2, "expected watcher and scraper tasks")
	assert.Equal(t, "trigger-watcher", tasks[0].Name)
	assert.Equal(t, "trigger-scraper", tasks[1].Name)
}

type leaseBackendNoop struct{}

func (leaseBackendNoop) TryAcquire(ctx context.Context, id, clusterID string) (bool, error) {
	return false, nil
}
