-- +goose Up
-- +goose StatementBegin
-- Denormalize workflow_name and status onto test_workflow_executions so that
-- aggregations (totals) and ordered list queries do not need to join
-- test_workflows / test_workflow_results in the hot path. Both columns are
-- kept in sync via triggers on the source-of-truth tables.
ALTER TABLE test_workflow_executions
    ADD COLUMN workflow_name VARCHAR(255),
    ADD COLUMN status        VARCHAR(50);
-- +goose StatementEnd

-- +goose StatementBegin
-- Backfill workflow_name from the canonical "workflow" row in test_workflows.
UPDATE test_workflow_executions e
   SET workflow_name = w.name
  FROM test_workflows w
 WHERE w.execution_id = e.id
   AND w.workflow_type = 'workflow'
   AND e.workflow_name IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Backfill status from test_workflow_results.
UPDATE test_workflow_executions e
   SET status = r.status
  FROM test_workflow_results r
 WHERE r.execution_id = e.id
   AND e.status IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Trigger: keep test_workflow_executions.status in sync with test_workflow_results.status.
CREATE OR REPLACE FUNCTION sync_execution_status() RETURNS trigger AS $$
BEGIN
    UPDATE test_workflow_executions
       SET status = NEW.status
     WHERE id = NEW.execution_id
       AND status IS DISTINCT FROM NEW.status;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_sync_execution_status
AFTER INSERT OR UPDATE OF status ON test_workflow_results
FOR EACH ROW EXECUTE FUNCTION sync_execution_status();
-- +goose StatementEnd

-- +goose StatementBegin
-- Trigger: keep test_workflow_executions.workflow_name in sync with the
-- canonical "workflow" row in test_workflows. Only fires when the workflow_type
-- row is the source-of-truth one ('workflow'), not 'resolved_workflow'.
CREATE OR REPLACE FUNCTION sync_execution_workflow_name() RETURNS trigger AS $$
BEGIN
    IF NEW.workflow_type = 'workflow' THEN
        UPDATE test_workflow_executions
           SET workflow_name = NEW.name
         WHERE id = NEW.execution_id
           AND workflow_name IS DISTINCT FROM NEW.name;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_sync_execution_workflow_name
AFTER INSERT OR UPDATE OF name, workflow_type ON test_workflows
FOR EACH ROW EXECUTE FUNCTION sync_execution_workflow_name();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_sync_execution_workflow_name ON test_workflows;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_sync_execution_status ON test_workflow_results;
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS sync_execution_workflow_name();
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS sync_execution_status();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE test_workflow_executions
    DROP COLUMN IF EXISTS workflow_name,
    DROP COLUMN IF EXISTS status;
-- +goose StatementEnd
