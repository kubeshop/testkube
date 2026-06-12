// Package controller runs the Agent-side loop that pushes cluster-environment
// inventory to the Control Plane. The Agent is authoritative about what it sees
// in its cluster; the Control Plane caches a snapshot.
//
// Pushes fire on:
//   - startup
//   - periodic ticker (default 1h; covers RBAC drift and missed CRD events)
//   - debounced CRD informer events (see crd_watcher.go)
package controller

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/clusterdiscovery"
)

// ClusterResourcesPusher is the subset of the inventory gRPC client we need.
type ClusterResourcesPusher interface {
	PutClusterResources(ctx context.Context, resources []testkube.ClusterResource) error
}

// ClusterResourcesDiscoverer is the subset of pkg/clusterdiscovery.Discoverer we need.
type ClusterResourcesDiscoverer interface {
	List(ctx context.Context) ([]testkube.ClusterResource, error)
}

// ClusterResourcesController pushes watchable cluster GVKs to the Control Plane
// on startup, on a periodic interval, and (optionally) when an external
// Notifier signals a change. Safe to cancel via context.
type ClusterResourcesController struct {
	Discoverer ClusterResourcesDiscoverer
	Pusher     ClusterResourcesPusher
	Interval   time.Duration
	// Notifier is fired by external sources (e.g. a CRD informer) when
	// inventory may have changed. Multiple signals in quick succession are
	// coalesced into one push via Debounce.
	Notifier <-chan struct{}
	// Debounce is the quiet period after the last Notifier event before the
	// push fires. Defaults to 5s - long enough to coalesce a burst (e.g. helm
	// install of an operator that registers ten CRDs at once) without delaying
	// a single-CRD update too noticeably in the UI.
	Debounce time.Duration
	Log      *zap.SugaredLogger
}

// Run blocks until ctx is canceled. Errors are logged but not returned - a
// transient push failure should not kill the agent; the next tick will retry.
func (c *ClusterResourcesController) Run(ctx context.Context) error {
	interval := c.Interval
	if interval <= 0 {
		interval = time.Hour
	}
	debounce := c.Debounce
	if debounce <= 0 {
		debounce = 5 * time.Second
	}

	c.pushOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// debounceC starts nil so the select arm is inactive until the first
	// notifier event arms the timer. Subsequent events extend the timer.
	var debounceTimer *time.Timer
	var debounceC <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return nil
		case <-ticker.C:
			c.pushOnce(ctx)
		case <-c.Notifier:
			if debounceTimer == nil {
				debounceTimer = time.NewTimer(debounce)
				debounceC = debounceTimer.C
			} else if !debounceTimer.Stop() {
				// timer was about to fire; drain channel before reset.
				select {
				case <-debounceC:
				default:
				}
			}
			debounceTimer.Reset(debounce)
		case <-debounceC:
			debounceTimer = nil
			debounceC = nil
			c.pushOnce(ctx)
		}
	}
}

func (c *ClusterResourcesController) pushOnce(ctx context.Context) {
	resources, err := c.Discoverer.List(ctx)
	if err != nil {
		c.Log.Warnw("inventory: cluster discovery failed; skipping push", "error", err)
		return
	}
	watchable := resources[:0]
	for _, r := range resources {
		if r.CanWatch {
			watchable = append(watchable, r)
		}
	}
	if err := c.Pusher.PutClusterResources(ctx, watchable); err != nil {
		c.Log.Warnw("inventory: push cluster resources to CP failed", "error", err, "count", len(watchable))
		return
	}
	c.Log.Infow("inventory: pushed cluster resources snapshot to CP", "count", len(watchable))
}

var _ ClusterResourcesDiscoverer = (*clusterdiscovery.Discoverer)(nil)
