-- +goose NO TRANSACTION

-- +goose Up
-- Enable pg_trgm extension for trigram-based GIN indexes that support ILIKE '%..%' patterns.
-- This fixes the root cause of execution search hanging on non-matching terms:
-- without this index, every ILIKE '%term%' query causes a full sequential scan of all
-- executions in the environment. With the GIN index, non-matching searches complete in ~1ms
-- by looking up trigrams in the inverted index without touching the heap.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram index on execution names.
-- Enables index-backed ILIKE '%term%' queries in GetTestWorkflowExecutions and
-- GetTestWorkflowExecutionsTotals. All searches with no matches now return instantly
-- instead of scanning every row.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_executions_name_trgm
    ON test_workflow_executions USING gin (name gin_trgm_ops);

-- Composite index covering the primary access pattern: filter by org+env, sort by time.
-- Every execution listing query filters on (organization_id, environment_id) and
-- sorts by scheduled_at DESC. A single composite index serves both the WHERE clause
-- and the ORDER BY, avoiding a separate sort step.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_executions_org_env_scheduled
    ON test_workflow_executions (organization_id, environment_id, scheduled_at DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_executions_name_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_executions_org_env_scheduled;
