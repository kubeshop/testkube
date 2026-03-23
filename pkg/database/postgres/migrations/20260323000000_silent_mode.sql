-- +goose Up
-- +goose StatementBegin
ALTER TABLE test_workflow_executions
    ADD COLUMN silent_mode JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE test_workflow_executions
    DROP COLUMN silent_mode;
-- +goose StatementEnd
