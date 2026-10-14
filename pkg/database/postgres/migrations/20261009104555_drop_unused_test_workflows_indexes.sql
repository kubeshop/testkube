-- +goose NO TRANSACTION

-- +goose Up

-- The unique key on (execution_id, workflow_type) serves every lookup that these indexes could serve.
-- idx_test_workflows_execution_id is a prefix of that key, and no query filters on workflow_type alone.
-- The label filters read labels per execution, which a GIN index cannot serve.
-- Postgres updates every index on each write, so an unused index costs write time and disk space.
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflows_execution_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflows_workflow_type;
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflows_labels;

-- +goose Down
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflows_execution_id ON test_workflows(execution_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflows_workflow_type ON test_workflows(workflow_type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflows_labels ON test_workflows USING GIN (labels);
