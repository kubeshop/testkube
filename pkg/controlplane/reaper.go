package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// reapStaleDispatches fails executions that were handed to a runner which never
// reported back, neither accepting nor declining them.
//
// Without this they sit in STARTING forever: the dispatch query keeps offering
// them, every offer costs a round trip, and the user sees an execution that
// never runs and never fails while the control plane reports healthy.
//
// The lease in the dispatch query already bounds how often such a row is
// retried. This bounds how long it is retried for.
func (s *Server) reapStaleDispatches(ctx context.Context) {
	ticker := time.NewTicker(s.reaperInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reapOnce(ctx)
		}
	}
}

func (s *Server) reapOnce(ctx context.Context) {
	log := s.cfg.Logger
	if log == nil {
		return
	}

	// Bounded, so one tick cannot stall on a large backlog.
	ids, err := s.executionQuerier.StaleStarting(ctx, time.Now().Add(-s.startTimeout()), s.dispatchBatchSize())
	if err != nil {
		log.Errorw("failed to look for stale dispatched executions", "err", err)
		return
	}

	for _, id := range ids {
		// AcceptExecution refuses a finished execution, so a runner that surfaces
		// late is told to tear down rather than resurrecting this one.
		//
		// The control plane is the actor here, not the runner: nothing was
		// declined, the dispatch was simply never acknowledged.
		cause := abortCause{
			reason:  testkube.StartReasonUnknown,
			message: fmt.Sprintf("The runner did not acknowledge this execution within %s.", s.startTimeout()),
			actor:   testkube.StopActorControlPlane,
		}
		if err := s.abortExecution(ctx, id, cause); err != nil {
			log.Warnw("failed to fail a stale dispatched execution", "id", id, "err", err)
			continue
		}
		log.Infow("failed a dispatched execution the runner never acknowledged",
			"id", id, "startTimeout", s.startTimeout())
	}
}
