-- +goose Up
-- +goose StatementBegin
-- Add runtime column
ALTER TABLE test_workflow_executions ADD COLUMN runtime JSONB;

-- Create indexes
CREATE INDEX idx_test_workflow_executions_runtime ON test_workflow_executions USING GIN (runtime);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_test_workflow_executions_runtime;

ALTER TABLE test_workflow_executions DROP COLUMN runtime;
-- +goose StatementEnd
