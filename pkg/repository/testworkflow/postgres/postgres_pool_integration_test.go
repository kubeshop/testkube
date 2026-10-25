package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

// singleConnPool opens a second pool against the same database as the one given,
// limited to one connection.
//
// One connection is the whole point: a function that opens a transaction and
// then reads through the pool needs two at once, so it can never complete here.
// With the production default of max(4, NumCPU) the same bug needs four
// executions finishing at the same instant to show itself, which is why it
// reached production as an unexplained stall rather than a failing test.
//
// The config is copied from the live pool rather than re-parsed from its
// ConnString: PreparePostgresTestDatabase points the pool at the temporary
// database by setting ConnConfig.Database on the already-parsed config, so the
// connection string still names the database it was parsed from. Re-parsing it
// silently connects to that one instead, where no migration has run - which
// fails as a missing relation and looks nothing like the deadlock this is for.
func singleConnPool(t *testing.T, from *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()

	cfg := from.Config().Copy()
	cfg.MaxConns = 1

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func seedRunningExecution(t *testing.T, pool *pgxpool.Pool, ctx context.Context, orgID, envID, id, runnerID string) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO test_workflow_executions
		(id, organization_id, environment_id, name, namespace, number, runner_id, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'default', 1, $5, NOW(), NOW(), NOW())
	`, id, orgID, envID, id, runnerID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO test_workflow_results (execution_id, status, steps, initialization)
		VALUES ($1, 'running', '{}', '{}')
	`, id)
	require.NoError(t, err)
}

// FinishResultStrict and UpdateResultStrict must hold exactly one connection.
//
// Both open a transaction and then read the execution back. Reading it through
// the pool rather than the transaction meant each call held one connection for
// the transaction while waiting for a second, so once as many executions
// finished at once as the pool had connections, every one of them blocked
// forever holding a transaction open - and everything else, the dispatch poll,
// the cron enqueue, the lease check, the reaper, queued behind them against a
// database that was completely idle.
//
// A pool of one makes that deterministic: the old code cannot finish a single
// call, the fixed code is unaffected.
func TestResultStrictHoldsOneConnection_Integration(t *testing.T) {
	test.IntegrationTest(t)

	testDB, cleanup := testpostgres.PreparePostgresTestDatabase(t, "strict_pool")
	t.Cleanup(cleanup)

	const (
		orgID    = "test-org"
		envID    = "test-env"
		runnerID = "test-runner"
	)

	pool := singleConnPool(t, testDB.Pool)
	repo := NewPostgresRepository(pool, WithOrganizationID(orgID), WithEnvironmentID(envID))

	setup := context.Background()
	seedRunningExecution(t, testDB.Pool, setup, orgID, envID, "exec-finish", runnerID)
	seedRunningExecution(t, testDB.Pool, setup, orgID, envID, "exec-update", runnerID)

	// Prove the second pool is on the migrated database holding the seed before
	// anything else runs. Without this, a pool pointed at the wrong database
	// fails inside the calls below as a missing relation, which reads like a
	// broken fixture rather than what this test is actually about.
	var seeded int
	require.NoError(t, pool.QueryRow(setup,
		`SELECT count(*) FROM test_workflow_executions WHERE id = ANY($1)`,
		[]string{"exec-finish", "exec-update"},
	).Scan(&seeded))
	require.Equal(t, 2, seeded, "the single-connection pool must see the seeded executions")

	t.Run("FinishResultStrict", func(t *testing.T) {
		// The deadline is the assertion: on the old code this call never returns,
		// so the failure reads as a timeout rather than a wrong value.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		updated, err := repo.FinishResultStrict(ctx, "exec-finish", runnerID, &testkube.TestWorkflowResult{
			Status:     common.Ptr(testkube.PASSED_TestWorkflowStatus),
			FinishedAt: time.Now().UTC(),
		})

		require.NoError(t, err, "a single-connection pool must be enough; needing two deadlocks the agent")
		assert.True(t, updated)
	})

	t.Run("UpdateResultStrict", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		updated, err := repo.UpdateResultStrict(ctx, "exec-update", runnerID, &testkube.TestWorkflowResult{
			Status: common.Ptr(testkube.RUNNING_TestWorkflowStatus),
		})

		require.NoError(t, err, "a single-connection pool must be enough; needing two deadlocks the agent")
		assert.True(t, updated)
	})

	t.Run("the pool is left idle, with nothing holding a transaction open", func(t *testing.T) {
		// A connection still checked out here would mean a transaction was never
		// committed or rolled back - the state the stalled replicas were found in,
		// every connection "idle in transaction".
		assert.Zero(t, pool.Stat().AcquiredConns(),
			"a returned call must leave no connection checked out")
	})
}
