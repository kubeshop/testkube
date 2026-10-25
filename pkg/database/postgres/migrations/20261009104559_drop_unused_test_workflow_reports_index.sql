-- +goose NO TRANSACTION

-- +goose Up

-- No query reads through rep_order. It only orders the rows of one execution.
-- Postgres updates every index on each write, so an unused index costs write time and disk space.
DROP INDEX CONCURRENTLY IF EXISTS idx_test_workflow_reports_rep_order;

-- +goose Down
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_test_workflow_reports_rep_order ON test_workflow_reports(rep_order);
