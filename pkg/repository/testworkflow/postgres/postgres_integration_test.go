package postgres

import (
	"context"
	"fmt"
	"testing"

	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgresRepositoryGetExecutionsIntegration tests the full repository GetExecutions method with real PostgreSQL
func TestPostgresRepositoryGetExecutionsIntegration(t *testing.T) {
	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "repo_executions")
	defer cleanup()

	ctx := context.Background()

	// Create repository
	orgID := "test-org"
	envID := "test-env"
	repo := NewPostgresRepository(
		testDB.Pool,
		WithOrganizationID(orgID),
		WithEnvironmentID(envID),
	)

	// Insert test data
	workflowID := "test-workflow"

	// Insert test workflow
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (id, organization_id, environment_id, name, namespace, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, workflowID, orgID, envID, "test-workflow", "default")
	require.NoError(t, err)

	// Insert test executions with various tags, labels, and selectors
	execution1 := "exec-1"
	execution2 := "exec-2"
	execution3 := "exec-3"

	// Execution 1: with tags and labels
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions 
		(id, organization_id, environment_id, test_workflow_id, name, namespace, number, scheduled_at, created_at, updated_at, tags, labels)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW(), $8, $9)
	`, execution1, orgID, envID, workflowID, "exec-1", "default", int32(1),
		`{"env": "prod", "team": "backend"}`, `{"app": "myapp", "version": "1.0"}`)
	require.NoError(t, err)

	// Execution 2: with different tags and labels
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions 
		(id, organization_id, environment_id, test_workflow_id, name, namespace, number, scheduled_at, created_at, updated_at, tags, labels)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW(), $8, $9)
	`, execution2, orgID, envID, workflowID, "exec-2", "default", int32(2),
		`{"env": "dev", "owner": "alice"}`, `{"app": "myapp", "tier": "frontend"}`)
	require.NoError(t, err)

	// Execution 3: with some overlapping tags and labels
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions 
		(id, organization_id, environment_id, test_workflow_id, name, namespace, number, scheduled_at, created_at, updated_at, tags, labels)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW(), $8, $9)
	`, execution3, orgID, envID, workflowID, "exec-3", "default", int32(3),
		`{"env": "prod", "owner": "bob"}`, `{"app": "yourapp", "version": "2.0"}`)
	require.NoError(t, err)

	// Insert results for executions
	for _, execID := range []string{execution1, execution2, execution3} {
		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_execution_results 
			(execution_id, status, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
		`, execID, "passed")
		require.NoError(t, err)
	}

	t.Run("Test tag filtering via repository", func(t *testing.T) {
		filter := testworkflow.NewExecutionsFilter().
			WithTagSelector("env=prod")

		result, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should find 2 executions with env=prod tag")
	})

	t.Run("Test label filtering via repository", func(t *testing.T) {
		labelSelector := &testworkflow.LabelSelector{
			Or: []testworkflow.Label{
				{Key: "app", Value: StringPtr("myapp")},
			},
		}

		filter := testworkflow.NewExecutionsFilter().
			WithLabelSelector(labelSelector)

		result, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should find 2 executions with app=myapp label")
	})

	t.Run("Test selector filtering via repository", func(t *testing.T) {
		filter := testworkflow.NewExecutionsFilter().
			WithSelector("app=myapp")

		result, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should find 2 executions with app=myapp selector")
	})

	t.Run("Test combined filtering via repository", func(t *testing.T) {
		labelSelector := &testworkflow.LabelSelector{
			Or: []testworkflow.Label{
				{Key: "app", Value: StringPtr("myapp")},
			},
		}

		filter := testworkflow.NewExecutionsFilter().
			WithTagSelector("env=prod").
			WithLabelSelector(labelSelector)

		result, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, 1, len(result), "Should find 1 execution with env=prod tag and app=myapp label")
	})

	t.Run("Test label existence filtering via repository", func(t *testing.T) {
		exists := true
		labelSelector := &testworkflow.LabelSelector{
			Or: []testworkflow.Label{
				{Key: "version", Exists: &exists},
			},
		}

		filter := testworkflow.NewExecutionsFilter().
			WithLabelSelector(labelSelector)

		result, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should find 2 executions with 'version' label")
	})

	t.Run("Test label non-existence filtering via repository", func(t *testing.T) {
		exists := false
		labelSelector := &testworkflow.LabelSelector{
			Or: []testworkflow.Label{
				{Key: "deprecated", Exists: &exists},
			},
		}

		filter := testworkflow.NewExecutionsFilter().
			WithLabelSelector(labelSelector)

		result, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, 3, len(result), "Should find all 3 executions without 'deprecated' label")
	})
}

// TestPostgresRepositoryGetExecutionsTotalsIntegration tests the full repository GetExecutionsTotals method with real PostgreSQL
func TestPostgresRepositoryGetExecutionsTotalsIntegration(t *testing.T) {
	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "repo_executions_totals")
	defer cleanup()

	ctx := context.Background()

	// Create repository
	orgID := "test-org"
	envID := "test-env"
	repo := NewPostgresRepository(
		testDB.Pool,
		WithOrganizationID(orgID),
		WithEnvironmentID(envID),
	)

	// Insert test data
	workflowID := "test-workflow"

	// Insert test workflow
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (id, organization_id, environment_id, name, namespace, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, workflowID, orgID, envID, "test-workflow", "default")
	require.NoError(t, err)

	// Insert test executions with different statuses
	for i := 0; i < 5; i++ {
		execID := fmt.Sprintf("exec-%d", i)
		status := "passed"
		if i%2 == 0 {
			status = "failed"
		}

		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_executions 
			(id, organization_id, environment_id, test_workflow_id, name, namespace, number, scheduled_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW())
		`, execID, orgID, envID, workflowID, execID, "default", int32(i+1))
		require.NoError(t, err)

		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_execution_results 
			(execution_id, status, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
		`, execID, status)
		require.NoError(t, err)
	}

	t.Run("Test totals without filters via repository", func(t *testing.T) {
		filter := testworkflow.NewExecutionsFilter()

		result, err := repo.GetExecutionsTotals(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, int32(3), result.Passed, "Should have 3 passed executions")
		assert.Equal(t, int32(2), result.Failed, "Should have 2 failed executions")
		assert.Equal(t, int32(5), result.Results, "Should have 5 total executions")
	})

	t.Run("Test totals with status filter via repository", func(t *testing.T) {
		filter := testworkflow.NewExecutionsFilter().
			WithStatus("passed")

		result, err := repo.GetExecutionsTotals(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, int32(3), result.Passed, "Should have 3 passed executions")
		assert.Equal(t, int32(0), result.Failed, "Should have 0 failed executions")
		assert.Equal(t, int32(3), result.Results, "Should have 3 total executions")
	})
}

// StringPtr is a helper function to create string pointers
func StringPtr(s string) *string {
	return &s
}
