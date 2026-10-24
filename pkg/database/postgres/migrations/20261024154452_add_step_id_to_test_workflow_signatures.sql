-- +goose Up
-- +goose StatementBegin
-- The stable id of the step. The ref of a step is random for each compile of the workflow, and the
-- id comes from the workflow, so a reader can match a result to a step by the id.
ALTER TABLE test_workflow_signatures ADD COLUMN IF NOT EXISTS step_id VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE test_workflow_signatures DROP COLUMN IF EXISTS step_id;
-- +goose StatementEnd
