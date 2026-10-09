-- +goose NO TRANSACTION

-- +goose Up

-- No query reads through out_order. It only orders the rows of one execution.
-- Postgres updates every index on each write, so an unused index costs write time and disk space.
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_outputs_out_order;

-- +goose Down
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_outputs_out_order ON test_workflow_outputs(out_order);
