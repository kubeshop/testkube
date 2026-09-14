-- +goose NO TRANSACTION

-- +goose Up
-- Lineage: where a rerun came from.
--
-- Scalars rather than a JSONB blob like `rerun`, because these are queried:
-- "every rerun of root R, in attempt order" is a range scan over the index
-- below, which a GIN containment index could neither order nor compose with
-- the organization/environment prefix every other query on this table uses.
--
-- Left NULL for rows written before lineage existed. Those rows mean exactly
-- what an original run means, and TestWorkflowExecution.EffectiveLineage()
-- synthesizes that, so no backfill rewrite of this table is needed - each
-- ADD COLUMN is nullable with no default and so is metadata-only.
--
-- The whole migration runs outside a transaction because the index has to be
-- built CONCURRENTLY: test_workflow_executions is populated and hot, and a
-- plain CREATE INDEX would hold a lock that blocks execution writes for the
-- length of the table scan, stalling scheduling during the rollout. That means
-- statements commit one by one, so every one of them is written to be
-- re-runnable after a failure partway through.
ALTER TABLE test_workflow_executions ADD COLUMN IF NOT EXISTS lineage_base_id TEXT;
ALTER TABLE test_workflow_executions ADD COLUMN IF NOT EXISTS lineage_root_id TEXT;
ALTER TABLE test_workflow_executions ADD COLUMN IF NOT EXISTS lineage_attempt INTEGER;

-- Backs "every execution of chain R, in attempt order": an ordered range scan
-- over the tenancy prefix, so the planner can walk the index and stop rather
-- than sorting a match set.
--
-- Indexed on COALESCE(lineage_root_id, id) rather than the column, and not
-- partial, because a row written before lineage existed carries NULL there and
-- is the root of its own chain. A predicate on the bare column would miss it,
-- so a rerun of a legacy execution would be findable while the original it
-- descends from would not. Chain queries must use the same COALESCE expression
-- for the index to apply.
--
-- The alternative is backfilling every legacy row to its own root, which
-- rewrites this whole table; the expression costs an index over all rows
-- instead, which is the cheaper of the two.
--
-- Drop first to heal an INVALID index left behind by an interrupted
-- CREATE INDEX CONCURRENTLY run; IF NOT EXISTS would otherwise skip it and
-- leave the retry stuck on an index that can never become valid.
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_lineage_root;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_twe_lineage_root
    ON test_workflow_executions (organization_id, environment_id, COALESCE(lineage_root_id, id), lineage_attempt);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_twe_lineage_root;

ALTER TABLE test_workflow_executions DROP COLUMN IF EXISTS lineage_attempt;
ALTER TABLE test_workflow_executions DROP COLUMN IF EXISTS lineage_root_id;
ALTER TABLE test_workflow_executions DROP COLUMN IF EXISTS lineage_base_id;
