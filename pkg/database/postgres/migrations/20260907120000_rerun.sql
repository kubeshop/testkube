-- +goose Up
-- +goose StatementBegin
-- Add rerun column: the test case selection an execution was narrowed to.
-- Recorded on the execution because that is how it reaches the runner - the
-- scheduler writes it, the runner reads it back when it starts the pod.
ALTER TABLE test_workflow_executions ADD COLUMN rerun JSONB;

-- Only reruns carry one, so the index covers the rows that have it rather than
-- the whole table.
CREATE INDEX idx_test_workflow_executions_rerun ON test_workflow_executions USING GIN (rerun) WHERE rerun IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_test_workflow_executions_rerun;

ALTER TABLE test_workflow_executions DROP COLUMN rerun;
-- +goose StatementEnd
