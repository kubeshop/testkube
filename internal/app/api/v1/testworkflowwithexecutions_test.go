package v1

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestListTestWorkflowWithExecutionsHandler_SortingLogic(t *testing.T) {
	t.Run("should sort by StartedAt with StatusAt fallback", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)
		
		// Create test workflows with different execution times
		workflows := []testkube.TestWorkflow{
			{Name: "workflow-old", Created: baseTime.Add(-15 * time.Minute)},
			{Name: "workflow-recent", Created: baseTime.Add(-12 * time.Minute)},
			{Name: "workflow-queued", Created: baseTime.Add(-6 * time.Minute)},
		}
		
		// Create executions with different timing scenarios
		executions := map[string]testkube.TestWorkflowExecutionSummary{
			"workflow-old": {
				Result: &testkube.TestWorkflowResultSummary{
					StartedAt: baseTime.Add(-10 * time.Minute), // Started 10 minutes ago
				},
				StatusAt: baseTime.Add(-8 * time.Minute), // Status 8 minutes ago
			},
			"workflow-recent": {
				Result: &testkube.TestWorkflowResultSummary{
					StartedAt: baseTime.Add(-5 * time.Minute), // Started 5 minutes ago
				},
				StatusAt: baseTime.Add(-3 * time.Minute), // Status 3 minutes ago
			},
			"workflow-queued": {
				Result: &testkube.TestWorkflowResultSummary{
					StartedAt: time.Time{}, // Zero time - not started
				},
				StatusAt: baseTime.Add(-2 * time.Minute), // Status 2 minutes ago (should use this)
			},
		}
		
		// Create results as would be done in the handler
		results := make([]testkube.TestWorkflowWithExecutionSummary, len(workflows))
		for i, workflow := range workflows {
			if execution, ok := executions[workflow.Name]; ok {
				results[i] = testkube.TestWorkflowWithExecutionSummary{
					Workflow:        &workflow,
					LatestExecution: &execution,
				}
			} else {
				results[i] = testkube.TestWorkflowWithExecutionSummary{
					Workflow: &workflow,
				}
			}
		}
		
		// Apply the same sorting logic as in the handler
		sortWorkflowExecutions(results)
		
		// Verify sorting order:
		// 1. workflow-queued: StatusAt = 2 minutes ago (most recent, uses fallback)
		// 2. workflow-recent: StartedAt = 5 minutes ago  
		// 3. workflow-old: StartedAt = 10 minutes ago
		assert.Equal(t, "workflow-queued", results[0].Workflow.Name, "Workflow with most recent StatusAt should be first")
		assert.Equal(t, "workflow-recent", results[1].Workflow.Name, "Workflow with recent StartedAt should be second")
		assert.Equal(t, "workflow-old", results[2].Workflow.Name, "Workflow with oldest StartedAt should be third")
	})

	t.Run("should fallback to workflow creation time when no executions", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)
		
		workflows := []testkube.TestWorkflow{
			{Name: "workflow-old", Created: baseTime.Add(-10 * time.Minute)},
			{Name: "workflow-new", Created: baseTime.Add(-2 * time.Minute)},
		}
		
		results := make([]testkube.TestWorkflowWithExecutionSummary, len(workflows))
		for i, workflow := range workflows {
			results[i] = testkube.TestWorkflowWithExecutionSummary{
				Workflow: &workflow,
			}
		}
		
		// Apply sorting logic
		sortWorkflowExecutions(results)
		
		// Newer workflow should be first
		assert.Equal(t, "workflow-new", results[0].Workflow.Name, "Newer workflow should be first")
		assert.Equal(t, "workflow-old", results[1].Workflow.Name, "Older workflow should be second")
	})

	t.Run("should prioritize StartedAt over StatusAt when both are present", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)
		
		workflows := []testkube.TestWorkflow{
			{Name: "workflow-a", Created: baseTime.Add(-15 * time.Minute)},
			{Name: "workflow-b", Created: baseTime.Add(-15 * time.Minute)},
		}
		
		// Workflow A: Started earlier but status updated more recently
		// Workflow B: Started later but status updated earlier
		// StartedAt should take precedence
		executions := map[string]testkube.TestWorkflowExecutionSummary{
			"workflow-a": {
				Result: &testkube.TestWorkflowResultSummary{
					StartedAt: baseTime.Add(-10 * time.Minute), // Started 10 minutes ago
				},
				StatusAt: baseTime.Add(-2 * time.Minute), // Status updated 2 minutes ago
			},
			"workflow-b": {
				Result: &testkube.TestWorkflowResultSummary{
					StartedAt: baseTime.Add(-5 * time.Minute), // Started 5 minutes ago (more recent)
				},
				StatusAt: baseTime.Add(-8 * time.Minute), // Status updated 8 minutes ago
			},
		}
		
		results := make([]testkube.TestWorkflowWithExecutionSummary, len(workflows))
		for i, workflow := range workflows {
			if execution, ok := executions[workflow.Name]; ok {
				results[i] = testkube.TestWorkflowWithExecutionSummary{
					Workflow:        &workflow,
					LatestExecution: &execution,
				}
			}
		}
		
		sortWorkflowExecutions(results)
		
		// workflow-b should be first because it started more recently (StartedAt takes precedence)
		assert.Equal(t, "workflow-b", results[0].Workflow.Name, "Workflow with more recent StartedAt should be first")
		assert.Equal(t, "workflow-a", results[1].Workflow.Name, "Workflow with older StartedAt should be second")
	})
}

func TestWorkflowExecutionSortingConsistency(t *testing.T) {
	t.Run("should have consistent sorting behavior", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)
		
		// Test data that should produce a specific order
		testCases := []struct {
			name        string
			startedAt   time.Time
			statusAt    time.Time
			created     time.Time
			expectedPos int // Expected position in sorted list (0 = first)
		}{
			{
				name:        "workflow-most-recent", 
				startedAt:   baseTime.Add(-1 * time.Minute),
				statusAt:    baseTime.Add(-30 * time.Second),
				created:     baseTime.Add(-5 * time.Minute),
				expectedPos: 0, // Should be first (most recent StartedAt)
			},
			{
				name:        "workflow-queued-recent",
				startedAt:   time.Time{}, // Zero time (not started)
				statusAt:    baseTime.Add(-2 * time.Minute),
				created:     baseTime.Add(-6 * time.Minute),
				expectedPos: 1, // Should be second (fallback to StatusAt)
			},
			{
				name:        "workflow-no-execution",
				startedAt:   time.Time{}, // No execution
				statusAt:    time.Time{}, // No execution
				created:     baseTime.Add(-3 * time.Minute),
				expectedPos: 2, // Should be third (fallback to Created)
			},
			{
				name:        "workflow-older",
				startedAt:   baseTime.Add(-4 * time.Minute),
				statusAt:    baseTime.Add(-3 * time.Minute),
				created:     baseTime.Add(-7 * time.Minute),
				expectedPos: 3, // Should be last (older StartedAt)
			},
		}
		
		// Create test data
		results := make([]testkube.TestWorkflowWithExecutionSummary, len(testCases))
		for i, tc := range testCases {
			workflow := testkube.TestWorkflow{
				Name:    tc.name,
				Created: tc.created,
			}
			
			result := testkube.TestWorkflowWithExecutionSummary{
				Workflow: &workflow,
			}
			
			// Add execution if we have timing data
			if !tc.startedAt.IsZero() || !tc.statusAt.IsZero() {
				result.LatestExecution = &testkube.TestWorkflowExecutionSummary{
					Result: &testkube.TestWorkflowResultSummary{
						StartedAt: tc.startedAt,
					},
					StatusAt: tc.statusAt,
				}
			}
			
			results[i] = result
		}
		
		// Apply sorting
		sortWorkflowExecutions(results)
		
		// Verify each workflow is in its expected position
		for _, tc := range testCases {
			found := false
			for j, result := range results {
				if result.Workflow.Name == tc.name {
					assert.Equal(t, tc.expectedPos, j, 
						"Workflow %s should be at position %d, but found at position %d", 
						tc.name, tc.expectedPos, j)
					found = true
					break
				}
			}
			assert.True(t, found, "Workflow %s not found in results", tc.name)
		}
		
		// Verify the overall order
		expectedOrder := []string{
			"workflow-most-recent",
			"workflow-queued-recent", 
			"workflow-no-execution",
			"workflow-older",
		}
		
		for i, expectedName := range expectedOrder {
			assert.Equal(t, expectedName, results[i].Workflow.Name, 
				"Position %d should have workflow %s", i, expectedName)
		}
	})
}

// Helper function that extracts the sorting logic from the handler for testing
func sortWorkflowExecutions(results []testkube.TestWorkflowWithExecutionSummary) {
	// This replicates the exact sorting logic from ListTestWorkflowWithExecutionsHandler
	sort.Slice(results, func(i, j int) bool {
		iTime := results[i].Workflow.Created
		if results[i].LatestExecution != nil {
			iTime = results[i].LatestExecution.Result.StartedAt
			// Fallback to StatusAt if StartedAt is not set (execution hasn't started yet)
			if iTime.IsZero() {
				iTime = results[i].LatestExecution.StatusAt
			}
		}
		jTime := results[j].Workflow.Created
		if results[j].LatestExecution != nil {
			jTime = results[j].LatestExecution.Result.StartedAt
			// Fallback to StatusAt if StartedAt is not set (execution hasn't started yet)
			if jTime.IsZero() {
				jTime = results[j].LatestExecution.StatusAt
			}
		}
		return iTime.After(jTime)
	})
}
