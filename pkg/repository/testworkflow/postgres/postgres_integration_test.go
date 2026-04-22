package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

// TestPostgresRepositoryGetExecutionsIntegration tests the full repository GetExecutions method with real PostgreSQL
func TestPostgresRepositoryGetExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)
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
	var err error

	// Insert test executions with various tags, labels, and selectors
	execution1 := "exec-1"
	execution2 := "exec-2"
	execution3 := "exec-3"

	// Execution 1: with tags
	_, err = testDB.Pool.Exec(ctx, `
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
func TestPostgresRepositoryGetExecutionsTotals_Integration(t *testing.T) {
	test.IntegrationTest(t)
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

	t.Run("Test totals without filters via repository", func(t *testing.T) {
		filter := testworkflow.NewExecutionsFilter()

		result, err := repo.GetExecutionsTotals(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, int32(2), result.Passed, "Should have 2 passed executions")
		assert.Equal(t, int32(3), result.Failed, "Should have 3 failed executions")
		assert.Equal(t, int32(5), result.Results, "Should have 5 total executions")
	})

	t.Run("Test totals with status filter via repository", func(t *testing.T) {
		filter := testworkflow.NewExecutionsFilter().
			WithStatus("passed")

		result, err := repo.GetExecutionsTotals(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, int32(2), result.Passed, "Should have 2 passed executions")
		assert.Equal(t, int32(0), result.Failed, "Should have 0 failed executions")
		assert.Equal(t, int32(2), result.Results, "Should have 2 total executions")
	})
}

// TestPostgresRepositoryCountExecutions_Integration tests the CountTestWorkflowExecutions method
// specifically verifying that label_keys and label_conditions work together with AND logic
func TestPostgresRepositoryCountExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)
	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "repo_count_executions")
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

	// Insert test executions with workflows having different labels
	// Execution 1: has label "app=myapp" and "version=1.0"
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions
		(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
	`, "exec-1", orgID, envID, "exec-1", "default", int32(1))
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_results (execution_id, status, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, "exec-1", "passed")
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, labels, created, updated)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, "exec-1", "workflow", "wf-1", "default", `{"app": "myapp", "version": "1.0"}`)
	require.NoError(t, err)

	// Execution 2: has label "app=myapp" but NO "version" label
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions
		(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
	`, "exec-2", orgID, envID, "exec-2", "default", int32(2))
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_results (execution_id, status, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, "exec-2", "passed")
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, labels, created, updated)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, "exec-2", "workflow", "wf-2", "default", `{"app": "myapp"}`)
	require.NoError(t, err)

	// Execution 3: has label "version=2.0" but "app=otherapp"
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_executions
		(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
	`, "exec-3", orgID, envID, "exec-3", "default", int32(3))
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflow_results (execution_id, status, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, "exec-3", "passed")
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, labels, created, updated)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, "exec-3", "workflow", "wf-3", "default", `{"app": "otherapp", "version": "2.0"}`)
	require.NoError(t, err)

	t.Run("Count with label existence AND value condition - should use AND logic", func(t *testing.T) {
		// This test verifies the bug fix: label_keys and label_conditions should use AND
		// We want: workflows that have "version" label AND "app=myapp"
		// Expected: Only exec-1 matches (has version label AND app=myapp)
		// If OR was used, exec-2 would also match (has app=myapp but no version)
		exists := true
		labelSelector := &testworkflow.LabelSelector{
			Or: []testworkflow.Label{
				{Key: "version", Exists: &exists},       // label_keys
				{Key: "app", Value: StringPtr("myapp")}, // label_conditions
			},
		}

		filter := testworkflow.NewExecutionsFilter().
			WithLabelSelector(labelSelector)

		count, err := repo.Count(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should count only 1 execution with both 'version' label AND 'app=myapp'")
	})

	t.Run("Count with only label existence filter", func(t *testing.T) {
		exists := true
		labelSelector := &testworkflow.LabelSelector{
			Or: []testworkflow.Label{
				{Key: "version", Exists: &exists},
			},
		}

		filter := testworkflow.NewExecutionsFilter().
			WithLabelSelector(labelSelector)

		count, err := repo.Count(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count, "Should count 2 executions with 'version' label")
	})

	t.Run("Count with only label value filter", func(t *testing.T) {
		labelSelector := &testworkflow.LabelSelector{
			Or: []testworkflow.Label{
				{Key: "app", Value: StringPtr("myapp")},
			},
		}

		filter := testworkflow.NewExecutionsFilter().
			WithLabelSelector(labelSelector)

		count, err := repo.Count(ctx, *filter)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count, "Should count 2 executions with app=myapp")
	})
}

// StringPtr is a helper function to create string pointers
func StringPtr(s string) *string {
	return &s
}

func TestPostgresDenormalizedExecutionColumns_Integration(t *testing.T) {
	test.IntegrationTest(t)
	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "repo_denorm_status_name")
	t.Cleanup(cleanup)

	ctx := context.Background()

	orgID := "test-org"
	envID := "test-env"
	repo := NewPostgresRepository(
		testDB.Pool,
		WithOrganizationID(orgID),
		WithEnvironmentID(envID),
	)

	insertExecution := func(t *testing.T, id string, num int32) {
		t.Helper()
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_executions
			(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
		`, id, orgID, envID, id, "default", num)
		require.NoError(t, err)
	}

	readDenormalized := func(t *testing.T, id string) (workflowName, status *string) {
		t.Helper()
		err := testDB.Pool.QueryRow(ctx,
			`SELECT workflow_name, status FROM test_workflow_executions WHERE id = $1`,
			id,
		).Scan(&workflowName, &status)
		require.NoError(t, err)
		return
	}

	t.Run("status trigger populates on insert into test_workflow_results", func(t *testing.T) {
		insertExecution(t, "exec-status-insert", 1)

		// Before any result exists, status is NULL.
		_, status := readDenormalized(t, "exec-status-insert")
		assert.Nil(t, status, "status should be NULL before any test_workflow_results row exists")

		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_results (execution_id, status, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
		`, "exec-status-insert", "queued")
		require.NoError(t, err)

		_, status = readDenormalized(t, "exec-status-insert")
		require.NotNil(t, status)
		assert.Equal(t, "queued", *status)
	})

	t.Run("status trigger updates on test_workflow_results.status change", func(t *testing.T) {
		insertExecution(t, "exec-status-update", 2)
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_results (execution_id, status, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
		`, "exec-status-update", "running")
		require.NoError(t, err)

		_, err = testDB.Pool.Exec(ctx,
			`UPDATE test_workflow_results SET status = $1 WHERE execution_id = $2`,
			"passed", "exec-status-update",
		)
		require.NoError(t, err)

		_, status := readDenormalized(t, "exec-status-update")
		require.NotNil(t, status)
		assert.Equal(t, "passed", *status)
	})

	t.Run("workflow_name trigger populates on insert with workflow_type='workflow'", func(t *testing.T) {
		insertExecution(t, "exec-name-insert", 3)

		name, _ := readDenormalized(t, "exec-name-insert")
		assert.Nil(t, name, "workflow_name should be NULL before any test_workflows row exists")

		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, created, updated)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, "exec-name-insert", "workflow", "my-workflow", "default")
		require.NoError(t, err)

		name, _ = readDenormalized(t, "exec-name-insert")
		require.NotNil(t, name)
		assert.Equal(t, "my-workflow", *name)
	})

	t.Run("workflow_name trigger ignores workflow_type='resolved_workflow'", func(t *testing.T) {
		insertExecution(t, "exec-name-resolved", 4)

		// Insert only the resolved_workflow row — should not populate workflow_name.
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, created, updated)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, "exec-name-resolved", "resolved_workflow", "resolved-name", "default")
		require.NoError(t, err)

		name, _ := readDenormalized(t, "exec-name-resolved")
		assert.Nil(t, name, "workflow_name must not be populated by resolved_workflow rows")

		// Now insert the canonical 'workflow' row — workflow_name should appear.
		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, created, updated)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, "exec-name-resolved", "workflow", "canonical-name", "default")
		require.NoError(t, err)

		name, _ = readDenormalized(t, "exec-name-resolved")
		require.NotNil(t, name)
		assert.Equal(t, "canonical-name", *name)
	})

	t.Run("workflow_name trigger updates when canonical row's name changes", func(t *testing.T) {
		insertExecution(t, "exec-name-update", 5)
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, created, updated)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, "exec-name-update", "workflow", "old-name", "default")
		require.NoError(t, err)

		_, err = testDB.Pool.Exec(ctx,
			`UPDATE test_workflows SET name = $1 WHERE execution_id = $2 AND workflow_type = 'workflow'`,
			"new-name", "exec-name-update",
		)
		require.NoError(t, err)

		name, _ := readDenormalized(t, "exec-name-update")
		require.NotNil(t, name)
		assert.Equal(t, "new-name", *name)
	})

	t.Run("GetExecutionsTotals reads denormalized columns and filters by workflow_name", func(t *testing.T) {
		// Set up two distinct workflows with mixed statuses.
		// wf-a: 2 passed, 1 failed
		// wf-b: 1 passed, 2 failed
		cases := []struct {
			id       string
			num      int32
			workflow string
			status   string
		}{
			{"tot-a-1", 100, "wf-a", "passed"},
			{"tot-a-2", 101, "wf-a", "passed"},
			{"tot-a-3", 102, "wf-a", "failed"},
			{"tot-b-1", 103, "wf-b", "passed"},
			{"tot-b-2", 104, "wf-b", "failed"},
			{"tot-b-3", 105, "wf-b", "failed"},
		}
		for _, c := range cases {
			insertExecution(t, c.id, c.num)
			_, err := testDB.Pool.Exec(ctx, `
				INSERT INTO test_workflow_results (execution_id, status, created_at, updated_at)
				VALUES ($1, $2, NOW(), NOW())
			`, c.id, c.status)
			require.NoError(t, err)
			_, err = testDB.Pool.Exec(ctx, `
				INSERT INTO test_workflows (execution_id, workflow_type, name, namespace, created, updated)
				VALUES ($1, $2, $3, $4, NOW(), NOW())
			`, c.id, "workflow", c.workflow, "default")
			require.NoError(t, err)
		}

		// Sanity: triggers populated all rows.
		var nullCount int
		require.NoError(t, testDB.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM test_workflow_executions
			WHERE id LIKE 'tot-%' AND (status IS NULL OR workflow_name IS NULL)
		`).Scan(&nullCount))
		assert.Equal(t, 0, nullCount, "all denormalized columns should be populated by triggers")

		// Filter by workflow_name — must match what the trigger wrote into e.workflow_name.
		filterA := testworkflow.NewExecutionsFilter().WithName("wf-a")
		resultA, err := repo.GetExecutionsTotals(ctx, *filterA)
		require.NoError(t, err)
		assert.Equal(t, int32(2), resultA.Passed, "wf-a should have 2 passed")
		assert.Equal(t, int32(1), resultA.Failed, "wf-a should have 1 failed")
		assert.Equal(t, int32(3), resultA.Results, "wf-a should have 3 total")

		filterB := testworkflow.NewExecutionsFilter().WithName("wf-b")
		resultB, err := repo.GetExecutionsTotals(ctx, *filterB)
		require.NoError(t, err)
		assert.Equal(t, int32(1), resultB.Passed, "wf-b should have 1 passed")
		assert.Equal(t, int32(2), resultB.Failed, "wf-b should have 2 failed")
		assert.Equal(t, int32(3), resultB.Results, "wf-b should have 3 total")
	})
}
