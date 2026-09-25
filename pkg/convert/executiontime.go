package convert

import (
	"math"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// This file repairs execution scheduling timestamps that legacy records lack.
// Executions created before the control plane became source of truth never had
// a ScheduledAt, so a straight copy persists a zero value that renders as the
// Unix epoch and corrupts every duration derived from it.
//
// Ported from testkube-cloud-api/pkg/executiontime, which the cloud convert
// tool applies to every execution for the same reason.

// minValidTime is the cutoff below which a stored timestamp is considered
// absent rather than real: zero time.Time values round-trip through MongoDB as
// 0001-01-01 and through some encodings as the Unix epoch.
var minValidTime = time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC)

// isUsableTime reports whether t is a real timestamp rather than a zero/epoch
// placeholder.
func isUsableTime(t time.Time) bool {
	return t.After(minValidTime)
}

// synthesizeScheduledAt returns the earliest usable timestamp carried by the
// execution, which is the best available lower bound for when it was
// scheduled. Using the minimum keeps ScheduledAt <= QueuedAt <= StartedAt <=
// FinishedAt, so durations derived from it can never go negative. It returns
// the zero time when the execution carries no usable timestamps at all.
func synthesizeScheduledAt(exec *testkube.TestWorkflowExecution) time.Time {
	anchors := []time.Time{exec.AssignedAt, exec.StatusAt}
	if exec.Result != nil {
		anchors = append(anchors, exec.Result.QueuedAt, exec.Result.StartedAt, exec.Result.FinishedAt)
	}

	var earliest time.Time
	for _, t := range anchors {
		if !isUsableTime(t) {
			continue
		}
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	return earliest
}

// repairExecutionTime fills a missing ScheduledAt from the execution's other
// timestamps and recomputes the duration fields wherever they cannot be
// trusted. Durations derived from a zero ScheduledAt overflow int32
// milliseconds and wrap to arbitrary values, so they are recomputed whenever
// the anchor was missing, and additionally whenever they are already negative
// (a wrap that persisted while the anchor was missing only in memory). It
// reports whether the execution was modified.
func repairExecutionTime(exec *testkube.TestWorkflowExecution) bool {
	if !isUsableTime(exec.ScheduledAt) {
		scheduledAt := synthesizeScheduledAt(exec)
		if scheduledAt.IsZero() {
			return false
		}
		exec.ScheduledAt = scheduledAt
		healDurations(exec.Result, scheduledAt)
		return true
	}

	if exec.Result != nil && (exec.Result.DurationMs < 0 || exec.Result.TotalDurationMs < 0) {
		return healDurations(exec.Result, exec.ScheduledAt)
	}
	return false
}

// healDurations recomputes the persisted duration fields against the repaired
// scheduling anchor. It intentionally does not use
// TestWorkflowResult.HealDuration: that rebuilds PausedMs from the Pauses
// list, which drops paused time on legacy records carrying only the counter.
// PausedMs is preserved as stored.
func healDurations(result *testkube.TestWorkflowResult, scheduledAt time.Time) bool {
	if result == nil || result.Status == nil || !result.Status.Finished() || !isUsableTime(result.FinishedAt) {
		return false
	}

	total := result.FinishedAt.Sub(scheduledAt)
	if total < 0 {
		total = 0
	}

	duration := total - time.Duration(result.PausedMs)*time.Millisecond
	if duration < 0 {
		duration = 0
	}

	result.TotalDurationMs = clampMs(total)
	result.DurationMs = clampMs(duration)
	result.TotalDuration = total.Round(time.Millisecond).String()
	result.Duration = duration.Round(time.Millisecond).String()
	return true
}

func clampMs(d time.Duration) int32 {
	if ms := d.Milliseconds(); ms <= math.MaxInt32 {
		return int32(ms)
	}
	return math.MaxInt32
}
