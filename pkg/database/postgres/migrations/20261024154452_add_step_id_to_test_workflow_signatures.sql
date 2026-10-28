-- +goose Up
-- +goose StatementBegin
-- The stable id of the step. The ref of a step is random for each compile of the workflow, and the
-- id comes from the workflow, so a reader can match a result to a step by the id. The runner sets no
-- limit on the length of an id, so the column sets none too.
ALTER TABLE test_workflow_signatures ADD COLUMN IF NOT EXISTS step_id TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE test_workflow_signatures DROP COLUMN IF EXISTS step_id;
-- +goose StatementEnd
