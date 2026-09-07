-- +goose Up
-- +goose StatementBegin
-- Add the non-passing test cases a report named, so they can be listed or
-- re-run without downloading the report file.
--
-- Capped by the writer, with failures_truncated marking a partial list: the
-- report file the row points at always holds the whole story.
ALTER TABLE test_workflow_reports ADD COLUMN failures JSONB;
ALTER TABLE test_workflow_reports ADD COLUMN failures_truncated BOOLEAN NOT NULL DEFAULT FALSE;

-- Only reports with failures carry the column, so the index covers those rows
-- rather than the whole table.
CREATE INDEX idx_test_workflow_reports_failures ON test_workflow_reports USING GIN (failures) WHERE failures IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_test_workflow_reports_failures;

ALTER TABLE test_workflow_reports DROP COLUMN failures_truncated;
ALTER TABLE test_workflow_reports DROP COLUMN failures;
-- +goose StatementEnd
