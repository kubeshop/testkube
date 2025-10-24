-- name: GetNextExecution :one
SELECT
    sqlc.embed(e),
    sqlc.embed(r)
FROM
    test_workflow_executions e
        JOIN test_workflow_results r ON e.id = r.execution_id
WHERE
    r.status IS NULL
   OR r.status IN ('queued', 'assigned', 'starting')
ORDER BY
    e.scheduled_at
LIMIT
    1
FOR UPDATE;

-- name: AssignExecutionRoot :one
UPDATE test_workflow_executions
SET
    runner_id = @runner_id::text,
    status_at = @ts::timestamptz,
    assigned_at = @ts::timestamptz,
    updated_at = @ts::timestamptz
WHERE id = @execution_id::text RETURNING *;

-- name: AssignExecutionResult :one
UPDATE test_workflow_results
SET
    status = 'assigned',
    updated_at = @ts::timestamptz
WHERE execution_id = @execution_id::text RETURNING *;


-- name: GetExecutionWorkflow :one
SELECT *
FROM test_workflows
WHERE workflow_type = 'workflow' AND execution_id = @execution_id::text;

-- name: GetExecutionResolvedWorkflow :one
SELECT *
FROM test_workflows
WHERE workflow_type = 'resolved_workflow' AND execution_id = @execution_id::text;

-- name: GetExecutionSignatures :many
SELECT *
FROM test_workflow_signatures
WHERE execution_id = @execution_id::text;

-- name: GetExecutionReports :many
SELECT *
FROM test_workflow_reports
WHERE execution_id = @execution_id::text;

-- name: GetExecutionOutputs :many
SELECT *
FROM test_workflow_outputs
WHERE execution_id = @execution_id::text;

-- name: GetExecutionAggregation :one
SELECT *
FROM test_workflow_resource_aggregations
WHERE execution_id = @execution_id::text;
