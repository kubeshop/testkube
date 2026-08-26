package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// countBatch backs the "Batches committed" statistic. A commit that only moves
// the checkpoint past documents that could not be migrated carries no rows, and
// reporting it as a batch would overstate the work done.
func TestCountBatchIgnoresRowlessCommits(t *testing.T) {
	t.Parallel()

	t.Run("a batch carrying rows counts", func(t *testing.T) {
		t.Parallel()
		stats := newStats(TaskExecutions)
		countBatch(stats, &batchCounts{executions: 5})
		assert.Equal(t, int64(1), stats.Batches)
	})

	t.Run("a checkpoint-only commit does not", func(t *testing.T) {
		t.Parallel()
		stats := newStats(TaskExecutions)
		countBatch(stats, &batchCounts{})
		assert.Zero(t, stats.Batches)
	})

	t.Run("child rows alone do not count as a batch", func(t *testing.T) {
		t.Parallel()
		// Child counts cannot be non-zero without a parent execution, so only the
		// execution count decides. Pinned so the predicate is not loosened to an
		// any-rows check, which would count a rowless commit as soon as some
		// unrelated counter moved.
		stats := newStats(TaskExecutions)
		countBatch(stats, &batchCounts{signatures: 3, workflows: 2})
		assert.Zero(t, stats.Batches)
	})
}
