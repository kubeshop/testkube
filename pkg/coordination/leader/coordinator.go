package leader

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
)

// Task represents a unit of work that should run only while this process holds the leader lease.
type Task struct {
	Name  string
	Start func(context.Context) error
}

// Option configures Coordinator behaviour.
type Option func(*Coordinator)

type taskGroup struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Coordinator coordinates leadership among multiple replicas using a lease backend. When a lease is
// acquired the coordinator starts all registered tasks with a derived context. If the lease is lost it
// cancels those contexts and waits for the tasks to exit before attempting to re-acquire leadership.
type Coordinator struct {
	backend    leasebackend.Repository
	identifier string
	clusterID  string
	logger     *zap.SugaredLogger

	checkInterval time.Duration

	mu     sync.Mutex
	tasks  []Task
	active *taskGroup
	leader bool
}

const (
	defaultCheckInterval = 5 * time.Second
)

// New creates a new Coordinator that uses the provided lease backend, identifier, and clusterID to
// participate in leader election.
func New(
	backend leasebackend.Repository,
	identifier string,
	clusterID string,
	logger *zap.SugaredLogger,
	options ...Option,
) *Coordinator {
	c := &Coordinator{
		backend:       backend,
		identifier:    identifier,
		clusterID:     clusterID,
		logger:        logger,
		checkInterval: defaultCheckInterval,
	}

	for _, opt := range options {
		opt(c)
	}

	if c.logger == nil {
		c.logger = zap.NewNop().Sugar()
	}

	return c
}

// WithCheckInterval overrides how often the coordinator revalidates or renews its lease.
func WithCheckInterval(interval time.Duration) Option {
	return func(c *Coordinator) {
		if interval > 0 {
			c.checkInterval = interval
		}
	}
}

// Register adds a task that must only run while this instance holds the leader lease. Register must be
// called before Run.
func (c *Coordinator) Register(task Task) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tasks = append(c.tasks, task)
}

// Run participates in leader election until the provided context is cancelled. It returns ctx.Err() when
// shutting down gracefully.
func (c *Coordinator) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	c.evaluate(ctx)

	for {
		select {
		case <-ctx.Done():
			c.release()
			return ctx.Err()
		case <-ticker.C:
			c.evaluate(ctx)
		}
	}
}

func (c *Coordinator) evaluate(ctx context.Context) {
	leased, err := c.backend.TryAcquire(ctx, c.identifier, c.clusterID)
	if err != nil {
		c.logger.Errorw("leader coordinator: failed to check lease", "error", err)
		// Treat errors as a lost lease to avoid duplicate work until we can revalidate.
		leased = false
	}

	if leased {
		c.acquire(ctx)
	} else {
		c.release()
	}
}

func (c *Coordinator) acquire(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.leader {
		return
	}

	c.logger.Infow("leader coordinator: acquired lease", "identifier", c.identifier, "clusterId", c.clusterID)

	childCtx, cancel := context.WithCancel(ctx)
	group := &taskGroup{cancel: cancel}

	for _, task := range c.tasks {
		if task.Start == nil {
			continue
		}

		task := task
		group.wg.Add(1)
		go func() {
			defer group.wg.Done()
			if err := task.Start(childCtx); err != nil && !errors.Is(err, context.Canceled) {
				c.logger.Errorw("leader coordinator: task exited with error", "task", task.Name, "error", err)
			}
		}()
	}

	c.active = group
	c.leader = true
}

func (c *Coordinator) release() {
	c.mu.Lock()
	group := c.active
	if !c.leader {
		c.mu.Unlock()
		return
	}
	c.leader = false
	c.active = nil
	c.mu.Unlock()

	c.logger.Infow("leader coordinator: releasing lease", "identifier", c.identifier, "clusterId", c.clusterID)

	if group == nil {
		c.logger.Warnw("leader coordinator: release called without active task group", "identifier", c.identifier, "clusterId", c.clusterID)
		return
	}

	group.cancel()
	group.wg.Wait()
}
