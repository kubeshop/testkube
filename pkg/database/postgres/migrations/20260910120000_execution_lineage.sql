-- +goose Up
-- +goose StatementBegin
-- Add lineage columns: where a rerun came from.
--
-- Scalars rather than a JSONB blob like `rerun`, because these are queried:
-- "every rerun of root R, in attempt order" is a range scan over the composite
-- index below, which a GIN containment index could neither order nor compose
-- with the organization/environment prefix every other query on this table uses.
-- `rerun` is a delivery channel nobody filters by; this is a dimension.
--
-- Left NULL for rows written before lineage existed. Those rows mean exactly
-- what an original run means, and TestWorkflowExecution.EffectiveLineage()
-- synthesizes that, so no backfill rewrite of this table is needed.
ALTER TABLE test_workflow_executions ADD COLUMN lineage_base_id TEXT;
ALTER TABLE test_workflow_executions ADD COLUMN lineage_root_id TEXT;
ALTER TABLE test_workflow_executions ADD COLUMN lineage_attempt INTEGER;

CREATE INDEX idx_twe_lineage_root
  ON test_workflow_executions (organization_id, environment_id, lineage_root_id, lineage_attempt)
  WHERE lineage_root_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_twe_lineage_root;

ALTER TABLE test_workflow_executions DROP COLUMN lineage_attempt;
ALTER TABLE test_workflow_executions DROP COLUMN lineage_root_id;
ALTER TABLE test_workflow_executions DROP COLUMN lineage_base_id;
-- +goose StatementEnd
