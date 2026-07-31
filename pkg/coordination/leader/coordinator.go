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
// Lease checks that fail against an unreachable backend are tolerated for one lease duration, so a
// transient outage does not restart every task.
type Coordinator struct {
	backend    leasebackend.Repository
	identifier string
	clusterID  string
	logger     *zap.SugaredLogger

	checkInterval time.Duration
	leaseDuration time.Duration
	now           func() time.Time

	mu                  sync.Mutex
	tasks               []Task
	active              *taskGroup
	leader              bool
	disabled            bool
	lastRenewal         time.Time
	consecutiveFailures int
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
		leaseDuration: leasebackend.DefaultMaxLeaseDuration,
		now:           time.Now,
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

// WithLeaseDuration overrides how long a successfully renewed lease stays valid. It must match the
// duration the lease backend writes, since it is also the window during which the coordinator keeps
// leadership while lease checks fail. Defaults to leasebackend.DefaultMaxLeaseDuration.
func WithLeaseDuration(duration time.Duration) Option {
	return func(c *Coordinator) {
		if duration > 0 {
			c.leaseDuration = duration
		}
	}
}

// withClock overrides the time source, so that tests can exercise the lease duration window
// without waiting for it.
func withClock(now func() time.Time) Option {
	return func(c *Coordinator) {
		if now != nil {
			c.now = now
		}
	}
}

// WithLeaderElectionDisabled controls whether leader election runs. When disabled,
// Run starts the registered tasks directly and never contacts the lease backend (no
// lease object, no periodic API calls). It is intended for explicitly single-replica
// deployments where election is unnecessary overhead.
//
// WARNING: do not disable this with more than one replica — every replica would run
// the leader-only tasks simultaneously.
func WithLeaderElectionDisabled(disabled bool) Option {
	return func(c *Coordinator) {
		c.disabled = disabled
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
	// Leader election disabled: run the registered tasks directly as the sole owner,
	// without ever contacting the lease backend. acquire()/release() handle task
	// lifecycle and never touch the backend, so this adds zero lease/API load.
	if c.disabled {
		c.logger.Infow("leader coordinator: leader election disabled, running tasks directly without lease coordination")
		c.acquire(ctx)
		<-ctx.Done()
		c.release()
		return ctx.Err()
	}

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
	// Timestamp the attempt before making it: backends stamp the renewal when the call starts, and a
	// throttled backend can take seconds to answer. Measuring the tolerance window from the response
	// would keep leadership past the point where other instances already treat the lease as expired.
	startedAt := c.now()

	leased, err := c.backend.TryAcquire(ctx, c.identifier, c.clusterID)
	if err != nil {
		c.handleCheckFailure(err)
		return
	}

	c.mu.Lock()
	c.consecutiveFailures = 0
	if leased {
		c.lastRenewal = startedAt
	}
	c.mu.Unlock()

	if !leased {
		c.release()
		return
	}

	c.acquire(ctx)
}

// handleCheckFailure decides whether a lease backend error costs us leadership. An unreachable
// backend does not mean the lease expired: it stays ours until the last successful renewal plus
// the lease duration, and only past that is leadership released.
func (c *Coordinator) handleCheckFailure(err error) {
	c.mu.Lock()
	c.consecutiveFailures++
	failures := c.consecutiveFailures
	isLeader := c.leader
	sinceRenewal := c.now().Sub(c.lastRenewal)
	c.mu.Unlock()

	if !isLeader {
		c.logger.Errorw("leader coordinator: failed to check lease",
			"error", err,
			"consecutiveFailures", failures)
		return
	}

	if sinceRenewal < c.leaseDuration {
		c.logger.Warnw("leader coordinator: failed to check lease, keeping leadership until the lease duration elapses",
			"error", err,
			"consecutiveFailures", failures,
			"sinceLastRenewal", sinceRenewal.String(),
			"leaseDuration", c.leaseDuration.String())
		return
	}

	c.logger.Errorw("leader coordinator: lease not renewed within the lease duration, releasing leadership",
		"error", err,
		"consecutiveFailures", failures,
		"sinceLastRenewal", sinceRenewal.String(),
		"leaseDuration", c.leaseDuration.String())

	c.release()
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
