package main

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/executioncache/volume"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// stepCacheLeaseInterval is how often a running execution's inbox is kept alive.
//
// Well under Sweeper.LeaseTTL below, so that one failed pass - an API server blip, a
// rollout - cannot expose a live inbox to the sweep.
const stepCacheLeaseInterval = 5 * time.Minute

// stepCacheLeaseTTL is how long an inbox survives without a refresh.
const stepCacheLeaseTTL = 30 * time.Minute

// refreshStepCacheLeases keeps the inbox of every running execution from being swept.
//
// The set of live executions is read back from the cluster rather than tracked in
// memory, which is what makes it trustworthy: a registration that is never cleared -
// a crash, a lost watch, a replica that went away - would pin an inbox for good, while
// a listing simply stops returning what is no longer running.
//
// Each pass is best effort. A failure leaves the leases it did not reach to expire,
// and the next pass renews them; nothing here is worth failing an agent over.
func refreshStepCacheLeases(ctx context.Context, worker executionworkertypes.Worker, mountPath string) {
	running, err := worker.List(ctx, executionworkertypes.ListOptions{Finished: common.Ptr(false)})
	if err != nil {
		log.DefaultLogger.Warnw("could not list running executions to renew step cache leases; an execution quiet for longer than the lease may have its inbox swept while it still holds it", "error", err)
		return
	}

	// Several resources of one execution share the root's inbox, so the same name
	// arrives repeatedly - a parallel fan-out is the usual reason.
	seen := make(map[string]struct{}, len(running))
	for _, item := range running {
		root := item.Resource.EffectiveRootId()
		if root == "" {
			continue
		}
		inbox := volume.InboxFor(root)
		if _, done := seen[inbox]; done {
			continue
		}
		seen[inbox] = struct{}{}

		// A missing inbox is the ordinary case for an execution that predates the
		// volume being configured, or one already swept, so it is not worth a line
		// each time round.
		_ = volume.TouchLease(mountPath, inbox)
	}
}

// runStepCacheLeaseRenewal renews the leases on a ticker until the context is
// cancelled.
//
// The first pass is the caller's, and has to be: Sweeper.Run sweeps as soon as it
// starts, and an agent that has just taken over holds no leases yet, so every live
// inbox would look abandoned to it.
func runStepCacheLeaseRenewal(ctx context.Context, worker executionworkertypes.Worker, mountPath string) {
	ticker := time.NewTicker(stepCacheLeaseInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshStepCacheLeases(ctx, worker, mountPath)
		}
	}
}
