-- +goose NO TRANSACTION

-- +goose Up

-- Partial index backing GetTestWorkflowExecutionTags. That query filters
-- test_workflow_executions by (organization_id, environment_id) and only cares
-- about rows that actually carry tags. Tags are sparse: only a tiny fraction of
-- executions set them, so the plain (organization_id, environment_id) predicate
-- plus the existing full GIN index on tags (idx_test_workflow_executions_tags,
-- which cannot narrow by org/env) left the planner doing a sequential scan over
-- the whole heap. This partial btree indexes only the tagged rows, turning the
-- tag-listing lookup into a small index scan.
-- Drop first to heal an INVALID index left behind by an interrupted
-- CREATE INDEX CONCURRENTLY run; IF NOT EXISTS would otherwise skip it.
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_org_env_tagged;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twe_org_env_tagged
    ON test_workflow_executions (organization_id, environment_id)
    WHERE tags IS NOT NULL
      AND tags <> '{}'::jsonb
      AND jsonb_typeof(tags) = 'object';

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_org_env_tagged;
