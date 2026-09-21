package controlplane

import (
	"context"
	"time"
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
		// A late AcceptExecution would resurrect one of these, because Init sets
		// SCHEDULING unconditionally. The start timeout is orders of magnitude
		// longer than a healthy start, so that race is not reachable in practice -
		// do not shorten it without guarding Init on the current status.
		if err := s.abortExecution(ctx, id); err != nil {
			log.Warnw("failed to fail a stale dispatched execution", "id", id, "err", err)
			continue
		}
		log.Infow("failed a dispatched execution the runner never acknowledged",
			"id", id, "startTimeout", s.startTimeout())
	}
}
