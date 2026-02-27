-- +goose Up
-- +goose StatementBegin
-- Add soft-delete column to test_workflow_executions
ALTER TABLE test_workflow_executions ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- Partial index for efficient lookup of soft-deleted rows (reaper queries)
CREATE INDEX idx_test_workflow_executions_deleted_at
    ON test_workflow_executions(deleted_at)
    WHERE deleted_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_test_workflow_executions_deleted_at;
ALTER TABLE test_workflow_executions DROP COLUMN deleted_at;
-- +goose StatementEnd
