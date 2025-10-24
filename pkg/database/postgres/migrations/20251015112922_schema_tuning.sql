-- +goose Up
-- +goose StatementBegin
-- Drop created_at and updated_at columns
ALTER TABLE test_workflows DROP COLUMN created_at;
ALTER TABLE test_workflows DROP COLUMN updated_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE test_workflows ADD COLUMN created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
ALTER TABLE test_workflows ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
-- +goose StatementEnd
