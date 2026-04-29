-- +goose NO TRANSACTION

-- +goose Up
-- Composite index that covers totals (org/env/workflow_name/status) and lets
-- counts grouped by status come straight from the index.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twe_org_env_wfname_status
    ON test_workflow_executions (organization_id, environment_id, workflow_name, status);

-- Composite index that covers the summary list query (org/env/workflow_name +
-- ORDER BY scheduled_at DESC LIMIT N), letting the executor walk the index
-- backwards and stop at the page boundary.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twe_org_env_wfname_sched
    ON test_workflow_executions (organization_id, environment_id, workflow_name, scheduled_at DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_org_env_wfname_sched;
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_org_env_wfname_status;
