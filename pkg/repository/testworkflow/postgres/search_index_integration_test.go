package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

// TestExecutionSearchIndexes_Integration verifies that the trigram and composite indexes
// created by migration 20260406120000_execution_search_indexes.sql back the production
// ILIKE text-search query, turning a sequential scan into a sub-millisecond index lookup.
//
// Run with:
//
//	INTEGRATION=true API_POSTGRES_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test ./pkg/repository/testworkflow/postgres/... -run TestExecutionSearchIndexes_Integration -v
func TestExecutionSearchIndexes_Integration(t *testing.T) {
	test.IntegrationTest(t)

	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "search_indexes")
	defer cleanup()

	ctx := context.Background()
	orgID := "org-search-test"
	envID := "env-search-test"

	// Insert 20 executions with recognisable names so the text-search subtests
	// can also verify that matching rows ARE returned correctly.
	for i := 1; i <= 20; i++ {
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO test_workflow_executions
				(id, organization_id, environment_id, name, namespace, number, scheduled_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`,
			fmt.Sprintf("search-exec-%02d", i),
			orgID,
			envID,
			fmt.Sprintf("deploy-workflow-%02d", i),
			"default",
			int32(i),
			time.Now().Add(-time.Duration(i)*time.Minute),
		)
		require.NoError(t, err)
	}

	// ── schema fixtures ──────────────────────────────────────────────────────

	t.Run("pg_trgm extension is enabled", func(t *testing.T) {
		var count int
		err := testDB.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM pg_extension WHERE extname = 'pg_trgm'`,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "pg_trgm extension should be installed by migration")
	})

	t.Run("trigram GIN index exists", func(t *testing.T) {
		var count int
		err := testDB.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM pg_indexes
			WHERE tablename = 'test_workflow_executions'
			  AND indexname  = 'idx_test_workflow_executions_name_trgm'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "trigram GIN index should exist after migration")
	})

	t.Run("composite org_env_scheduled index exists", func(t *testing.T) {
		var count int
		err := testDB.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM pg_indexes
			WHERE tablename = 'test_workflow_executions'
			  AND indexname  = 'idx_test_workflow_executions_org_env_scheduled'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "composite index should exist after migration")
	})

	// ── query plan fixtures ──────────────────────────────────────────────────
	// We disable sequential scans so that PostgreSQL must use an index if one
	// covers the query; this is required for small test tables whose statistics
	// would otherwise favour a seq scan.

	// explainLines runs an EXPLAIN on a dedicated connection (so SET enable_seqscan
	// does not leak to the pool) and returns the full plan as a single string.
	explainLines := func(t *testing.T, sql string, args ...any) string {
		t.Helper()
		conn, err := testDB.Pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()

		_, err = conn.Exec(ctx, "SET enable_seqscan = off")
		require.NoError(t, err)

		rows, err := conn.Query(ctx, sql, args...)
		require.NoError(t, err)
		defer rows.Close()

		var lines []string
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			lines = append(lines, line)
		}
		require.NoError(t, rows.Err())
		plan := strings.Join(lines, "\n")
		t.Logf("Query plan:\n%s", plan)
		return plan
	}

	t.Run("non-matching ILIKE uses trigram index", func(t *testing.T) {
		plan := explainLines(t, `
			EXPLAIN SELECT id FROM test_workflow_executions
			WHERE organization_id = $1
			  AND environment_id  = $2
			  AND name ILIKE '%' || $3 || '%'
			ORDER BY scheduled_at DESC
			LIMIT 16
		`, orgID, envID, "nonexistent_xyz_abc_12345")

		assert.Contains(t, plan, "idx_test_workflow_executions_name_trgm",
			"plan should use the trigram index for ILIKE '%%..%%'")
		assert.NotContains(t, plan, "Seq Scan",
			"plan must not fall back to sequential scan")
	})

	t.Run("list by org+env uses composite index", func(t *testing.T) {
		plan := explainLines(t, `
			EXPLAIN SELECT id FROM test_workflow_executions
			WHERE organization_id = $1
			  AND environment_id  = $2
			ORDER BY scheduled_at DESC
			LIMIT 16
		`, orgID, envID)

		assert.Contains(t, plan, "idx_test_workflow_executions_org_env_scheduled",
			"plan should use the composite index for ORDER BY scheduled_at DESC")
		assert.NotContains(t, plan, "Seq Scan",
			"plan must not fall back to sequential scan")
	})

	// ── repository-level fixtures ────────────────────────────────────────────

	t.Run("non-matching text search returns empty results", func(t *testing.T) {
		repo := NewPostgresRepository(testDB.Pool,
			WithOrganizationID(orgID),
			WithEnvironmentID(envID),
		)
		filter := testworkflow.NewExecutionsFilter().WithTextSearch("nonexistent_xyz_abc_12345")
		results, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Empty(t, results, "non-matching text search should return 0 results immediately")
	})

	t.Run("matching text search returns correct executions", func(t *testing.T) {
		repo := NewPostgresRepository(testDB.Pool,
			WithOrganizationID(orgID),
			WithEnvironmentID(envID),
		)
		filter := testworkflow.NewExecutionsFilter().WithTextSearch("deploy-workflow")
		results, err := repo.GetExecutions(ctx, *filter)
		require.NoError(t, err)
		assert.Len(t, results, 20, "should find all 20 executions matching 'deploy-workflow'")
	})

	// ── timing fixture (smoke-level, not a hard benchmark) ──────────────────
	// Verifies the non-matching query completes well under typical timeout.
	t.Run("non-matching search completes within 100ms", func(t *testing.T) {
		conn, err := testDB.Pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()

		start := time.Now()
		_, err = conn.Exec(ctx,
			`SELECT id FROM test_workflow_executions
			 WHERE organization_id = $1
			   AND environment_id  = $2
			   AND name ILIKE '%' || $3 || '%'
			 ORDER BY scheduled_at DESC LIMIT 16`,
			orgID, envID, "nonexistent_xyz_abc_12345",
		)
		elapsed := time.Since(start)
		require.NoError(t, err)

		assert.Less(t, elapsed, 100*time.Millisecond,
			"non-matching ILIKE search should be sub-100ms with trigram index (got %s)", elapsed)
	})
}
