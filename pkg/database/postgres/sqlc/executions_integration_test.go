package sqlc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

// TestGetTestWorkflowExecutionsIntegration tests the GetTestWorkflowExecutions query with real PostgreSQL
func TestGetTestWorkflowExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)
	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "executions")
	defer cleanup()

	ctx := context.Background()
	queries := New(testDB.Pool)

	// Insert test data
	orgID := "test-org"
	envID := "test-env"

	// Insert test executions with various tags, labels, and selectors
	execution1 := "exec-1"
	execution2 := "exec-2"
	execution3 := "exec-3"

	// Execution 1: with tags
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions
		(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at, tags)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW(), $7)
	`, execution1, orgID, envID, "exec-1", "default", int32(1),
		`{"env": "prod", "team": "backend"}`)
	require.NoError(t, err)

	// Execution 2: with different tags
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions
		(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at, tags)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW(), $7)
	`, execution2, orgID, envID, "exec-2", "default", int32(2),
		`{"env": "dev", "owner": "alice"}`)
	require.NoError(t, err)

	// Execution 3: with some overlapping tags
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions
		(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at, tags)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW(), $7)
	`, execution3, orgID, envID, "exec-3", "default", int32(3),
		`{"env": "prod", "owner": "bob"}`)
	require.NoError(t, err)

	// Insert results for executions
	for _, execID := range []string{execution1, execution2, execution3} {
		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_results
			(execution_id, status, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
		`, execID, "passed")
		require.NoError(t, err)
	}

	// Insert test workflows for each execution with appropriate labels
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, labels, created, updated)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, execution1, "workflow", "test-workflow-1", "default", `{"app": "myapp", "version": "1.0"}`)
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, labels, created, updated)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, execution2, "workflow", "test-workflow-2", "default", `{"app": "myapp", "tier": "frontend"}`)
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, labels, created, updated)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, execution3, "workflow", "test-workflow-3", "default", `{"app": "yourapp", "version": "2.0"}`)
	require.NoError(t, err)

	t.Run("Test tag filtering with key existence", func(t *testing.T) {
		// Test filtering by tag key existence
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			TagKeys:        []string{"env"},
			Lmt:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result), "Should find all 3 executions with 'env' tag")
	})

	t.Run("Test tag filtering with key non-existence", func(t *testing.T) {
		// Test filtering by tag key non-existence
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			TagKeys:        []string{"deprecated:not_exists"},
			Lmt:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result), "Should find all 3 executions without 'deprecated' tag")
	})

	t.Run("Test tag filtering with key-value conditions", func(t *testing.T) {
		// Test filtering by tag key-value conditions
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			TagConditions:  []string{"env=prod"},
			Lmt:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should find 2 executions with env=prod")
	})

	t.Run("Test label filtering with key existence", func(t *testing.T) {
		// Test filtering by label key existence
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			LabelKeys:      []string{"app"},
			Lmt:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result), "Should find all 3 executions with 'app' label")
	})

	t.Run("Test label filtering with key non-existence", func(t *testing.T) {
		// Test filtering by label key non-existence
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			LabelKeys:      []string{"deprecated:not_exists"},
			Lmt:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result), "Should find all 3 executions without 'deprecated' label")
	})

	t.Run("Test label filtering with key-value conditions", func(t *testing.T) {
		// Test filtering by label key-value conditions
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID:  orgID,
			EnvironmentID:   envID,
			LabelConditions: []string{"app=myapp"},
			Lmt:             10,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should find 2 executions with app=myapp")
	})

	t.Run("Test selector filtering with key existence", func(t *testing.T) {
		// Test filtering by selector key existence
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			SelectorKeys:   []string{"app"},
			Lmt:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result), "Should find all 3 executions with 'app' selector")
	})

	t.Run("Test selector filtering with key non-existence", func(t *testing.T) {
		// Test filtering by selector key non-existence
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			SelectorKeys:   []string{"deprecated:not_exists"},
			Lmt:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result), "Should find all 3 executions without 'deprecated' selector")
	})

	t.Run("Test selector filtering with key-value conditions", func(t *testing.T) {
		// Test filtering by selector key-value conditions
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID:     orgID,
			EnvironmentID:      envID,
			SelectorConditions: []string{"app=myapp"},
			Lmt:                10,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should find 2 executions with app=myapp selector")
	})

	t.Run("Test combined filtering", func(t *testing.T) {
		// Test combining multiple filter types
		result, err := queries.GetTestWorkflowExecutions(ctx, GetTestWorkflowExecutionsParams{
			OrganizationID:  orgID,
			EnvironmentID:   envID,
			TagConditions:   []string{"env=prod"},
			LabelConditions: []string{"app=myapp"},
			Lmt:             10,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(result), "Should find 1 execution with env=prod tag and app=myapp label")
	})
}

// TestGetTestWorkflowExecutionsTotalsIntegration tests the GetTestWorkflowExecutionsTotals query with real PostgreSQL
func TestGetTestWorkflowExecutionsTotals_Integration(t *testing.T) {
	test.IntegrationTest(t)
	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "executions_totals")
	defer cleanup()

	ctx := context.Background()
	queries := New(testDB.Pool)

	// Insert test data
	orgID := "test-org"
	envID := "test-env"
	var err error

	// Insert test executions with different statuses
	for i := 0; i < 5; i++ {
		execID := fmt.Sprintf("exec-%d", i)
		status := "passed"
		if i%2 == 0 {
			status = "failed"
		}

		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_executions
			(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
		`, execID, orgID, envID, execID, "default", int32(i+1))
		require.NoError(t, err)

		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_results
			(execution_id, status, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
		`, execID, status)
		require.NoError(t, err)
	}

	// Insert test workflow for the first execution
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, created, updated)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, "exec-0", "workflow", "test-workflow", "default")
	require.NoError(t, err)

	t.Run("Test totals without filters", func(t *testing.T) {
		result, err := queries.GetTestWorkflowExecutionsTotals(ctx, GetTestWorkflowExecutionsTotalsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should have 2 status groups")

		// Find passed and failed counts
		passedCount := int64(0)
		failedCount := int64(0)
		for _, row := range result {
			switch row.Status.String {
			case "passed":
				passedCount = row.Count
			case "failed":
				failedCount = row.Count
			}
		}

		assert.Equal(t, int64(2), passedCount, "Should have 2 passed executions")
		assert.Equal(t, int64(3), failedCount, "Should have 3 failed executions")
		assert.Equal(t, int64(5), passedCount+failedCount, "Should have 5 total executions")
	})

	t.Run("Test totals with status filter", func(t *testing.T) {
		result, err := queries.GetTestWorkflowExecutionsTotals(ctx, GetTestWorkflowExecutionsTotalsParams{
			OrganizationID: orgID,
			EnvironmentID:  envID,
			Statuses:       []string{"passed"},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(result), "Should have 1 status group")
		assert.Equal(t, int64(2), result[0].Count, "Should have 2 passed executions")
	})
}
