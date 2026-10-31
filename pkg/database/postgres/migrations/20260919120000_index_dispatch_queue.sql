-- +goose NO TRANSACTION

-- +goose Up
-- Serves the runner dispatch query (GetExecutionsToStart). The pending set is
-- tiny next to the table, so a partial index lets Postgres walk it in
-- scheduled_at order and stop at the LIMIT instead of sorting every execution
-- ever scheduled.
-- Drop first to heal an INVALID index left behind by an interrupted
-- CREATE INDEX CONCURRENTLY run; IF NOT EXISTS would otherwise skip it.
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_dispatch_queue;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twe_dispatch_queue
    ON test_workflow_executions (status, scheduled_at)
    WHERE status IN ('assigned', 'starting');

-- Serves GetExecutionTransitions. idx_test_workflow_results_status already
-- covers the column, but it indexes every terminal row too; mid-transition rows
-- are a vanishing fraction of the table, so the partial index stays small enough
-- to remain cached.
DROP INDEX CONCURRENTLY IF EXISTS idx_twr_pending_transitions;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twr_pending_transitions
    ON test_workflow_results (status)
    WHERE status IN ('pausing', 'resuming', 'stopping');

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_dispatch_queue;
DROP INDEX CONCURRENTLY IF EXISTS idx_twr_pending_transitions;
