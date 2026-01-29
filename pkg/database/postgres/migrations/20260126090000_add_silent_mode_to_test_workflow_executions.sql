-- +goose Up
-- +goose StatementBegin
ALTER TABLE test_workflow_executions
ADD COLUMN IF NOT EXISTS silent_mode JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE test_workflow_executions
DROP COLUMN IF EXISTS silent_mode;
-- +goose StatementEnd


