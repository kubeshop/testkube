package sqlc_test

import (
	"context"
	"encoding/json"
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

// seedResult writes one execution and one result row that carries the given
// steps and initialization documents. It returns the execution id.
func seedResult(t *testing.T, db *database.DB, ctx context.Context, steps, initialization string) string {
	t.Helper()

	executionID := uuid.NewString()
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO test_workflow_executions (id, name) VALUES ($1, $2)`,
		executionID, "wf-"+executionID)
	require.NoError(t, err)

	_, err = db.Pool.Exec(ctx,
		`INSERT INTO test_workflow_results (execution_id, status, steps, initialization)
		 VALUES ($1, 'running', $2, $3)`,
		executionID, steps, initialization)
	require.NoError(t, err)

	return executionID
}

// TestForceCancelExecutionQueries_Integration is the first test of this
// package. It pins the four defects that the control plane copy of these
// queries already fixed: a COALESCE that mixed a text branch with a jsonb
// branch, a concatenation that let the stale timestamps win, a key spelled
// finishedaAt, and an aggregate that returned NULL over zero rows.
func TestForceCancelExecutionQueries_Integration(t *testing.T) {
	test.IntegrationTest(t)

	db, cleanup := testpostgres.PreparePostgresTestDatabase(t, "force_cancel_queries")
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlc.New(db.Pool)

	finishedAt := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	zeroTime := "0001-01-01T00:00:00Z"

	t.Run("steps cancel and keep every other key", func(t *testing.T) {
		executionID := seedResult(t,
			db, ctx,
			`{"active":{"status":"running","queuedAt":"`+zeroTime+`","errorMessage":"keep-me"},
			  "done":{"status":"passed","queuedAt":"2020-01-01T00:00:00Z"}}`,
			`{"status":"running"}`)

		require.NoError(t, queries.ForceCancelExecutionSteps(ctx, sqlc.ForceCancelExecutionStepsParams{
			ExecutionID: executionID,
			FinishedAt:  pgtype.Timestamptz{Time: finishedAt, Valid: true},
		}))

		steps := map[string]map[string]any{}
		require.NoError(t, json.Unmarshal(readSteps(t, db, ctx, executionID), &steps))

		assert.Equal(t, "canceled", steps["active"]["status"], "an active step must cancel")
		assert.Equal(t, "keep-me", steps["active"]["errorMessage"], "the other keys must survive")
		assert.NotEqual(t, zeroTime, steps["active"]["queuedAt"],
			"the new timestamp must win over the zero value")
		assert.Equal(t, "passed", steps["done"]["status"], "a terminated step must keep its status")
	})

	t.Run("an empty steps object stays an empty object", func(t *testing.T) {
		executionID := seedResult(t, db, ctx, `{}`, `{"status":"running"}`)

		require.NoError(t, queries.ForceCancelExecutionSteps(ctx, sqlc.ForceCancelExecutionStepsParams{
			ExecutionID: executionID,
			FinishedAt:  pgtype.Timestamptz{Time: finishedAt, Valid: true},
		}))

		raw := readSteps(t, db, ctx, executionID)
		require.NotNil(t, raw, "steps must stay a jsonb object, never SQL NULL")
		assert.JSONEq(t, `{}`, string(raw))
	})

	t.Run("initialization cancels and writes finishedAt", func(t *testing.T) {
		executionID := seedResult(t, db, ctx, `{}`,
			`{"status":"running","queuedAt":"`+zeroTime+`"}`)

		require.NoError(t, queries.ForceCancelExecutionInitialization(ctx, sqlc.ForceCancelExecutionInitializationParams{
			ExecutionID: executionID,
			FinishedAt:  pgtype.Timestamptz{Time: finishedAt, Valid: true},
		}))

		var raw []byte
		require.NoError(t, db.Pool.QueryRow(ctx,
			`SELECT initialization FROM test_workflow_results WHERE execution_id = $1`,
			executionID).Scan(&raw))

		initialization := map[string]any{}
		require.NoError(t, json.Unmarshal(raw, &initialization))

		assert.Equal(t, "canceled", initialization["status"])
		assert.Contains(t, initialization, "finishedAt", "the finish time must use the finishedAt key")
		assert.NotContains(t, initialization, "finishedaAt", "the misspelled key must not appear")
	})
}

func readSteps(t *testing.T, db *database.DB, ctx context.Context, executionID string) []byte {
	t.Helper()

	var raw []byte
	require.NoError(t, db.Pool.QueryRow(ctx,
		`SELECT steps FROM test_workflow_results WHERE execution_id = $1`,
		executionID).Scan(&raw))
	return raw
}
