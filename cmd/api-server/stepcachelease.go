package main

import (
	"context"
	"os"
	"time"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/executioncache/volume"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// defaultStepCacheRetention stands in for a retention that is not a positive number of
// days, which would otherwise disable the sweep and let the shared volume fill. It is
// the charts' own default, so the substitution lands where an unset value would have.
const defaultStepCacheRetention = 7 * 24 * time.Hour

// stepCacheLifecycleMargin allows for how much longer than its configured days an
// object store may go on serving a pointer.
//
// **It narrows the window rather than closing it, and cannot close it.** Nothing in the
// lifecycle contract bounds how late a removal may be, so a store slow enough will
// still be serving a pointer when the sweep takes the entry it names - and that key
// then answers every run with a hit that restores nothing, immutably, until the object
// finally goes. Closing it needs a positive signal that the object is gone, which this
// process cannot get: attached to a Control Plane it has no access to the bucket its
// pointers are written to. The fix is the Control Plane's - deleting or replacing a
// pointer when a restore reports its entry missing, or reporting removals an agent can
// act on - and until then this is a mitigation, chosen because erring long costs only
// an entry kept past its usefulness.
//
// A day-based lifecycle rule is not a deadline. S3, and the stores that follow it, add
// the days to the object's creation time and then round up to the next UTC midnight -
// so a one-day rule keeps an object for up to two days - and the removal after that is
// asynchronous, with no bound promised on the lag. Twenty-four hours covers the
// rounding and twenty-four more the removal, erring where the only cost is an entry
// kept past its usefulness rather than a key that misses until its object goes.
const stepCacheLifecycleMargin = 48 * time.Hour

// stepCachePublicationGrace covers the gap between committing an entry and storing the
// pointer that names it, which is what keeps retention genuinely longer than a
// pointer's life rather than merely equal to it. Generous on purpose: the toolkit's
// upload allows five attempts of up to thirty minutes, and the only cost of being wrong
// in this direction is an entry outliving its usefulness.
const stepCachePublicationGrace = 6 * time.Hour

// stepCacheLeaseInterval is how often a running execution's inbox is kept alive.
//
// Well under stepCacheLeaseTTL below, so that one failed pass - an API server blip, a
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
	var (
		failed     int
		firstErr   error
		firstInbox string
	)
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

		// A missing inbox is the ordinary case - an execution that predates the volume
		// being configured, or one already swept - and is not worth a line each time
		// round. Anything else is the protection failing silently: a volume remounted
		// read-only, a permission the agent lost, an I/O error. The lease then goes
		// stale while the execution is still running, and the sweep takes an inbox its
		// pod is still writing to, so an operator has to hear about it while there is
		// still time to act.
		if err := volume.TouchLease(mountPath, inbox); err != nil && !os.IsNotExist(err) {
			failed++
			if firstErr == nil {
				firstErr, firstInbox = err, inbox
			}
		}
	}

	// One line for the pass rather than one per execution: a volume that has gone
	// read-only fails every inbox, and a hundred identical lines every five minutes
	// buries the thing it is reporting.
	if failed > 0 {
		log.DefaultLogger.Errorw(
			"could not renew step cache leases; the inboxes of running executions may be swept while their pods still write to them, and the pointers they publish would then name entries that are gone",
			"failed", failed, "of", len(seen), "inbox", firstInbox, "error", firstErr)
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
