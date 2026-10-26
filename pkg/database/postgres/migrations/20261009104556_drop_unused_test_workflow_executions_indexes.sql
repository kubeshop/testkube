-- +goose NO TRANSACTION

-- +goose Up

-- No query uses a jsonb operator on running_context or runtime, and no query filters on namespace.
-- idx_twe_org_env_wfname_status and idx_twe_org_env_runner_status_sched serve the reads by organization.
-- Postgres updates every index on each write, so an unused index costs write time and disk space.
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_executions_running_context;
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_executions_runtime;
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_executions_namespace;
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_executions_organization_id;

-- +goose Down
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_executions_running_context ON test_workflow_executions USING GIN (running_context);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_executions_runtime ON test_workflow_executions USING GIN (runtime);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_executions_namespace ON test_workflow_executions(namespace);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_executions_organization_id ON test_workflow_executions(organization_id);
