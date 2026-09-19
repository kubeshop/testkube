package sqlc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
	database "github.com/kubeshop/testkube/pkg/database/postgres"
	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

// seedExecution writes one execution and its result row.
//
// It deliberately sets the status only on test_workflow_results: the dispatch
// query reads test_workflow_executions.status, which a trigger keeps in step, so
// seeding this way also proves that denormalisation works.
func seedExecution(tb testing.TB, db *database.DB, ctx context.Context, status string, scheduledAt, statusAt time.Time) string {
	tb.Helper()

	executionID := uuid.NewString()
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO test_workflow_executions (id, name, workflow_name, scheduled_at, status_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		executionID, "wf-a-"+executionID, "wf-a", scheduledAt, statusAt)
	require.NoError(tb, err)

	_, err = db.Pool.Exec(ctx,
		`INSERT INTO test_workflow_results (execution_id, status, steps, initialization)
		 VALUES ($1, $2, '{}', '{}')`,
		executionID, status)
	require.NoError(tb, err)

	return executionID
}

func ts(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// The dispatch read is bounded and ordered, which is the whole point: unbounded,
// one poll grew with the backlog until it exceeded the runner's call deadline.
func TestGetExecutionsToStart_Integration(t *testing.T) {
	test.IntegrationTest(t)

	db, cleanup := testpostgres.PreparePostgresTestDatabase(t, "executions_to_start")
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlc.New(db.Pool)
	now := time.Now().UTC()

	t.Run("returns the oldest scheduled first, bounded by the limit", func(t *testing.T) {
		var want []string
		for i := 0; i < 5; i++ {
			id := seedExecution(t, db, ctx, "assigned", now.Add(time.Duration(i)*time.Minute), now)
			want = append(want, id)
		}

		rows, err := queries.GetExecutionsToStart(ctx, sqlc.GetExecutionsToStartParams{
			RedispatchBefore: ts(now.Add(-time.Minute)),
			BatchSize:        3,
		})
		require.NoError(t, err)

		require.Len(t, rows, 3, "the batch size has to bound the read")
		for i, row := range rows {
			assert.Equal(t, want[i], row.TestWorkflowExecution.ID, "oldest scheduled first")
			assert.Equal(t, "assigned", row.TestWorkflowExecution.Status.String,
				"the denormalising trigger has to populate e.status, which this query reads")
			assert.Equal(t, "wf-a", row.TestWorkflowExecution.WorkflowName.String)
		}
	})

	t.Run("a starting row is withheld until its lease expires", func(t *testing.T) {
		db2, cleanup2 := testpostgres.PreparePostgresTestDatabase(t, "dispatch_lease")
		t.Cleanup(cleanup2)
		q2 := sqlc.New(db2.Pool)

		fresh := seedExecution(t, db2, ctx, "starting", now, now)
		stale := seedExecution(t, db2, ctx, "starting", now, now.Add(-5*time.Minute))

		rows, err := q2.GetExecutionsToStart(ctx, sqlc.GetExecutionsToStartParams{
			RedispatchBefore: ts(now.Add(-time.Minute)),
			BatchSize:        10,
		})
		require.NoError(t, err)

		require.Len(t, rows, 1, "only the expired lease may be re-offered")
		assert.Equal(t, stale, rows[0].TestWorkflowExecution.ID)
		assert.NotEqual(t, fresh, rows[0].TestWorkflowExecution.ID,
			"a freshly dispatched row must not be re-offered on the next poll")
	})

	t.Run("terminal rows are never offered", func(t *testing.T) {
		db3, cleanup3 := testpostgres.PreparePostgresTestDatabase(t, "dispatch_terminal")
		t.Cleanup(cleanup3)
		q3 := sqlc.New(db3.Pool)

		for _, status := range []string{"running", "passed", "failed", "aborted", "queued"} {
			seedExecution(t, db3, ctx, status, now, now.Add(-time.Hour))
		}

		rows, err := q3.GetExecutionsToStart(ctx, sqlc.GetExecutionsToStartParams{
			RedispatchBefore: ts(now),
			BatchSize:        10,
		})
		require.NoError(t, err)
		assert.Empty(t, rows)
	})
}

// A batch is claimed in one round trip, and only rows that are still assigned
// move.
func TestStartExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	db, cleanup := testpostgres.PreparePostgresTestDatabase(t, "start_executions")
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlc.New(db.Pool)
	now := time.Now().UTC()

	assignedA := seedExecution(t, db, ctx, "assigned", now, now.Add(-time.Hour))
	assignedB := seedExecution(t, db, ctx, "assigned", now, now.Add(-time.Hour))
	running := seedExecution(t, db, ctx, "running", now, now.Add(-time.Hour))

	ids := []string{assignedA, assignedB, running}
	require.NoError(t, queries.StartExecutions(ctx, sqlc.StartExecutionsParams{
		ExecutionIds: ids,
		StatusAt:     ts(now),
	}))
	require.NoError(t, queries.StartExecutionsResult(ctx, ids))

	statuses := map[string]string{}
	rows, err := db.Pool.Query(ctx,
		`SELECT execution_id, status FROM test_workflow_results WHERE execution_id = ANY($1)`, ids)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id, status string
		require.NoError(t, rows.Scan(&id, &status))
		statuses[id] = status
	}

	assert.Equal(t, "starting", statuses[assignedA])
	assert.Equal(t, "starting", statuses[assignedB])
	assert.Equal(t, "running", statuses[running], "a row that is already running must not be dragged back")
}

// Renewing the lease is what keeps the retry interval at the configured value
// instead of collapsing to the poll interval.
func TestRefreshStartingExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	db, cleanup := testpostgres.PreparePostgresTestDatabase(t, "refresh_starting")
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlc.New(db.Pool)
	now := time.Now().UTC()
	old := now.Add(-time.Hour)

	starting := seedExecution(t, db, ctx, "starting", now, old)
	assigned := seedExecution(t, db, ctx, "assigned", now, old)

	require.NoError(t, queries.RefreshStartingExecutions(ctx, sqlc.RefreshStartingExecutionsParams{
		ExecutionIds: []string{starting, assigned},
		StatusAt:     ts(now),
	}))

	var startingAt, assignedAt time.Time
	require.NoError(t, db.Pool.QueryRow(ctx,
		`SELECT status_at FROM test_workflow_executions WHERE id = $1`, starting).Scan(&startingAt))
	require.NoError(t, db.Pool.QueryRow(ctx,
		`SELECT status_at FROM test_workflow_executions WHERE id = $1`, assigned).Scan(&assignedAt))

	assert.WithinDuration(t, now, startingAt, time.Second, "the lease has to be renewed")
	assert.WithinDuration(t, old, assignedAt, time.Second, "only starting rows hold a dispatch lease")
}

func TestGetStaleStartingExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	db, cleanup := testpostgres.PreparePostgresTestDatabase(t, "stale_starting")
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlc.New(db.Pool)
	now := time.Now().UTC()

	stale := seedExecution(t, db, ctx, "starting", now, now.Add(-30*time.Minute))
	seedExecution(t, db, ctx, "starting", now, now)                // still within its timeout
	seedExecution(t, db, ctx, "running", now, now.Add(-time.Hour)) // not a dispatch at all

	ids, err := queries.GetStaleStartingExecutions(ctx, sqlc.GetStaleStartingExecutionsParams{
		StaleBefore: ts(now.Add(-10 * time.Minute)),
		BatchSize:   10,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{stale}, ids)
}

// The transitions read is seeded through the real producers, so this pins the
// actual producer/consumer pairing rather than a hand-written literal. It is the
// strongest guard against the ABORTED/CANCELLED mapping being swapped again.
func TestGetExecutionTransitions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	db, cleanup := testpostgres.PreparePostgresTestDatabase(t, "execution_transitions")
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlc.New(db.Pool)
	now := time.Now().UTC()

	canceled := seedExecution(t, db, ctx, "running", now, now)
	aborted := seedExecution(t, db, ctx, "running", now, now)
	paused := seedExecution(t, db, ctx, "running", now, now)
	seedExecution(t, db, ctx, "running", now, now) // stays running, must not appear

	require.NoError(t, queries.CancelExecutionRunningResult(ctx, canceled))
	require.NoError(t, queries.AbortExecutionRunningResult(ctx, aborted))
	require.NoError(t, queries.PauseExecutionResult(ctx, paused))

	rows, err := queries.GetExecutionTransitions(ctx, []string{"pausing", "resuming", "stopping"})
	require.NoError(t, err)

	got := map[string][2]string{}
	for _, row := range rows {
		got[row.ExecutionID] = [2]string{row.Status.String, row.PredictedStatus.String}
	}

	require.Len(t, got, 3, "only executions mid-transition are returned")
	assert.Equal(t, [2]string{"stopping", "canceled"}, got[canceled],
		"a user cancel has to be readable as a cancel, or the runner aborts instead")
	assert.Equal(t, [2]string{"stopping", "aborted"}, got[aborted],
		"a user abort has to be readable as an abort")
	assert.Equal(t, "pausing", got[paused][0])
}

// BenchmarkGetExecutionsToStart_Integration is the headline measurement: the
// cost of one poll against a growing backlog.
//
// Before the fix this read every pending row and hydrated each with six more
// queries, so ns/op and allocs/op grew linearly with depth. After it, they are
// flat past the batch size. Run it with `make bench-integration`.
func BenchmarkGetExecutionsToStart_Integration(b *testing.B) {
	test.IntegrationTest(b)

	ctx := context.Background()
	now := time.Now().UTC()

	for _, depth := range []int{10, 100, 1000, 5000} {
		b.Run(fmt.Sprintf("backlog=%d", depth), func(b *testing.B) {
			db, cleanup := testpostgres.PreparePostgresTestDatabase(b, fmt.Sprintf("bench_to_start_%d", depth))
			b.Cleanup(cleanup)
			queries := sqlc.New(db.Pool)

			for i := 0; i < depth; i++ {
				seedExecution(b, db, ctx, "assigned", now.Add(time.Duration(i)*time.Second), now)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows, err := queries.GetExecutionsToStart(ctx, sqlc.GetExecutionsToStartParams{
					RedispatchBefore: ts(now.Add(-time.Minute)),
					BatchSize:        25,
				})
				if err != nil {
					b.Fatal(err)
				}
				if len(rows) > 25 {
					b.Fatalf("the read is not bounded: got %d rows", len(rows))
				}
			}
		})
	}
}
