package convert

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// capture returns a logger plus the messages written through it.
func capture(t *testing.T) (*zap.SugaredLogger, func() []string) {
	t.Helper()

	core, logs := observer.New(zap.WarnLevel)
	return zap.New(core).Sugar(), func() []string {
		out := make([]string, 0, logs.Len())
		for _, e := range logs.All() {
			out = append(out, e.Message)
		}
		return out
	}
}

// Messages stop being retained past maxRetainedErrors, so len(Errors)
// understates the failures. Print must report the counter instead, or a badly
// damaged collection would look far healthier than it is.
func TestAddErrorRetainsBoundedMessages(t *testing.T) {
	t.Parallel()

	stats := newStats(TaskExecutions)
	const failures = maxRetainedErrors * 3
	for i := range failures {
		stats.addError("execution %d is broken", i)
	}

	assert.Equal(t, int64(failures), stats.Failed, "every failure must be counted")
	assert.Len(t, stats.Errors, maxRetainedErrors, "retained messages must be capped")
}

func TestPrintReportsFailedCountNotRetainedCount(t *testing.T) {
	t.Parallel()

	t.Run("more failures than are retained", func(t *testing.T) {
		t.Parallel()

		log, messages := capture(t)
		stats := newStats(TaskExecutions)
		for i := range maxRetainedErrors * 2 {
			stats.addError("execution %d is broken", i)
		}
		stats.finish()
		stats.Print(log, nil)

		out := messages()
		require.NotEmpty(t, out)

		// The headline is the true failure count, not the number of kept messages.
		assert.Contains(t, out, fmt.Sprintf("Encountered %d errors (showing %d):",
			maxRetainedErrors*2, maxReportedErrors))
		// And the omitted tail is counted against the true total.
		assert.Contains(t, out, fmt.Sprintf("... and %d more",
			maxRetainedErrors*2-maxReportedErrors))
	})

	t.Run("fewer failures than the report limit", func(t *testing.T) {
		t.Parallel()

		log, messages := capture(t)
		stats := newStats(TaskExecutions)
		stats.addError("only one broke")
		stats.finish()
		stats.Print(log, nil)

		out := messages()
		assert.Contains(t, out, "Encountered 1 errors (showing 1):")
		for _, m := range out {
			assert.NotContains(t, m, "and 0 more", "nothing was omitted, so say nothing")
		}
	})

	t.Run("no failures prints no error block", func(t *testing.T) {
		t.Parallel()

		log, messages := capture(t)
		stats := newStats(TaskExecutions)
		stats.finish()
		stats.Print(log, nil)

		for _, m := range messages() {
			assert.NotContains(t, m, "Encountered")
		}
	})
}

// Verification compares the whole source collection against the whole target
// table, so it reads the cumulative counters. A resumed run's own counters say
// nothing about what an earlier run declined.
func TestCumulativeCountersDriveExpectedRowCount(t *testing.T) {
	t.Parallel()

	// Eight documents, three of which no run can migrate. The second run has
	// nothing left to do, so its own Failed is zero while the cumulative total
	// still remembers the three.
	stats := &Stats{Task: TaskExecutions, Total: 8, Processed: 0, Failed: 0, CumulativeFailed: 3}

	expected := stats.Total - stats.CumulativeFailed - stats.CumulativeSkipped
	assert.Equal(t, int64(5), expected,
		"a resumed run must still expect only the migratable documents")

	perRun := stats.Total - stats.Failed - stats.Skipped
	assert.Equal(t, int64(8), perRun,
		"the per-run counters would have expected every document, which is the bug")
}
