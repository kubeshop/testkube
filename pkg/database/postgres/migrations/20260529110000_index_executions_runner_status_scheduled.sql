-- +goose NO TRANSACTION

-- +goose Up
-- Covers execution summary pages filtered by runner/status and ordered by
-- scheduled_at DESC, so Postgres can stop at the page boundary before
-- hydrating execution details.
-- Drop first to heal an INVALID index left behind by an interrupted
-- CREATE INDEX CONCURRENTLY run; IF NOT EXISTS would otherwise skip it.
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_org_env_runner_status_sched;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twe_org_env_runner_status_sched
    ON test_workflow_executions (organization_id, environment_id, runner_id, status, scheduled_at DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_org_env_runner_status_sched;
