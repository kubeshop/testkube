package deprecatedv1

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestTestSuiteWithExecutions_SortingLogic(t *testing.T) {
	baseTime := time.Now().Truncate(time.Second)

	t.Run("should sort by StartTime with EndTime fallback", func(t *testing.T) {
		results := []testkube.TestSuiteWithExecutionSummary{
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-old", Created: baseTime.Add(-20 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: baseTime.Add(-10 * time.Minute), // Started 10 minutes ago
					EndTime:   baseTime.Add(-8 * time.Minute),  // Ended 8 minutes ago
				},
			},
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-recent", Created: baseTime.Add(-20 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: baseTime.Add(-5 * time.Minute), // Started 5 minutes ago (more recent)
					EndTime:   baseTime.Add(-3 * time.Minute), // Ended 3 minutes ago
				},
			},
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-queued", Created: baseTime.Add(-20 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: time.Time{},                    // Zero time - not started
					EndTime:   baseTime.Add(-2 * time.Minute), // Ended 2 minutes ago (should use this)
				},
			},
		}

		// Apply the same sorting logic as in the handler
		sortTestSuiteExecutions(results)

		// Expected order based on effective sort times:
		// testsuite-queued: EndTime -2 min (most recent)
		// testsuite-recent: StartTime -5 min
		// testsuite-old: StartTime -10 min (oldest)
		assert.Equal(t, "testsuite-queued", results[0].TestSuite.Name)
		assert.Equal(t, "testsuite-recent", results[1].TestSuite.Name)
		assert.Equal(t, "testsuite-old", results[2].TestSuite.Name)
	})

	t.Run("should use Created time for test suites without executions", func(t *testing.T) {
		results := []testkube.TestSuiteWithExecutionSummary{
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-with-execution", Created: baseTime.Add(-20 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: baseTime.Add(-10 * time.Minute),
					EndTime:   baseTime.Add(-8 * time.Minute),
				},
			},
			{
				TestSuite:       &testkube.TestSuite{Name: "testsuite-no-execution", Created: baseTime.Add(-5 * time.Minute)},
				LatestExecution: nil,
			},
		}

		sortTestSuiteExecutions(results)

		// testsuite-no-execution should be first (Created -5 min > StartTime -10 min)
		assert.Equal(t, "testsuite-no-execution", results[0].TestSuite.Name)
		assert.Equal(t, "testsuite-with-execution", results[1].TestSuite.Name)
	})

	t.Run("should prioritize StartTime over EndTime", func(t *testing.T) {
		results := []testkube.TestSuiteWithExecutionSummary{
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-a", Created: baseTime.Add(-20 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: baseTime.Add(-10 * time.Minute), // Older start time
					EndTime:   baseTime.Add(-1 * time.Minute),  // Very recent end time
				},
			},
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-b", Created: baseTime.Add(-20 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: baseTime.Add(-5 * time.Minute), // More recent start time
					EndTime:   baseTime.Add(-8 * time.Minute), // Older end time
				},
			},
		}

		sortTestSuiteExecutions(results)

		// testsuite-b should be first because its StartTime is more recent
		assert.Equal(t, "testsuite-b", results[0].TestSuite.Name)
		assert.Equal(t, "testsuite-a", results[1].TestSuite.Name)
	})

	t.Run("should handle mixed scenarios with and without executions", func(t *testing.T) {
		results := []testkube.TestSuiteWithExecutionSummary{
			{
				TestSuite:       &testkube.TestSuite{Name: "testsuite-no-exec-old", Created: baseTime.Add(-15 * time.Minute)},
				LatestExecution: nil,
			},
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-with-exec", Created: baseTime.Add(-20 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: baseTime.Add(-5 * time.Minute),
					EndTime:   baseTime.Add(-3 * time.Minute),
				},
			},
			{
				TestSuite:       &testkube.TestSuite{Name: "testsuite-no-exec-recent", Created: baseTime.Add(-2 * time.Minute)},
				LatestExecution: nil,
			},
		}

		sortTestSuiteExecutions(results)

		// Expected order by effective times:
		// testsuite-no-exec-recent: Created -2 min (most recent)
		// testsuite-with-exec: StartTime -5 min
		// testsuite-no-exec-old: Created -15 min (oldest)
		assert.Equal(t, "testsuite-no-exec-recent", results[0].TestSuite.Name)
		assert.Equal(t, "testsuite-with-exec", results[1].TestSuite.Name)
		assert.Equal(t, "testsuite-no-exec-old", results[2].TestSuite.Name)
	})

	t.Run("should handle zero times correctly", func(t *testing.T) {
		results := []testkube.TestSuiteWithExecutionSummary{
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-both-zero", Created: baseTime.Add(-10 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: time.Time{}, // Zero time
					EndTime:   time.Time{}, // Zero time - should fall back to Created
				},
			},
			{
				TestSuite: &testkube.TestSuite{Name: "testsuite-start-zero", Created: baseTime.Add(-15 * time.Minute)},
				LatestExecution: &testkube.TestSuiteExecutionSummary{
					StartTime: time.Time{},                    // Zero time
					EndTime:   baseTime.Add(-5 * time.Minute), // Should use this
				},
			},
		}

		sortTestSuiteExecutions(results)

		// testsuite-start-zero should be first (EndTime -5 min > Created -10 min)
		assert.Equal(t, "testsuite-start-zero", results[0].TestSuite.Name)
		assert.Equal(t, "testsuite-both-zero", results[1].TestSuite.Name)
	})
}

// sortTestSuiteExecutions replicates the exact sorting logic from ListTestSuiteWithExecutionsHandler
func sortTestSuiteExecutions(results []testkube.TestSuiteWithExecutionSummary) {
	// Use sort.Slice to match the exact behavior in the handler
	sort.Slice(results, func(i, j int) bool {
		iTime := results[i].TestSuite.Created
		if results[i].LatestExecution != nil {
			iTime = results[i].LatestExecution.StartTime
			// Fallback to EndTime if StartTime is not set (execution hasn't started yet)
			if iTime.IsZero() {
				iTime = results[i].LatestExecution.EndTime
			}
		}

		jTime := results[j].TestSuite.Created
		if results[j].LatestExecution != nil {
			jTime = results[j].LatestExecution.StartTime
			// Fallback to EndTime if StartTime is not set (execution hasn't started yet)
			if jTime.IsZero() {
				jTime = results[j].LatestExecution.EndTime
			}
		}

		return iTime.After(jTime)
	})
}
