-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twe_status_pending_at_id
    ON test_workflow_executions (status, (COALESCE(status_at, scheduled_at)), id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_status_pending_at_id;
