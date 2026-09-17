-- +goose Up

-- statusDetails says why an execution did not pass. The runner sends it inside the result.
-- Postgres maps each result field to its own column, so the database drops a field without one.
-- Executions that ended before this column existed keep a null value. There is no backfill,
-- because the signals that the classifier reads are not stored for them.
ALTER TABLE test_workflow_results ADD COLUMN status_details JSONB;

COMMENT ON COLUMN test_workflow_results.status_details IS
    'Why the execution did not pass: type, reason, message, step, actor, and user. Null for an execution that passed, and for an execution that ended before this column existed.';

-- +goose Down
ALTER TABLE test_workflow_results DROP COLUMN status_details;
