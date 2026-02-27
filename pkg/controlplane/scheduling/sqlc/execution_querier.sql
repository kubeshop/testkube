-- name: GetExecutionsByStatus :many
SELECT
    sqlc.embed(e),
    sqlc.embed(r)
FROM
    test_workflow_executions e
        JOIN test_workflow_results r ON e.id = r.execution_id
WHERE r.status = @status::text
  AND (COALESCE(@predicted_status::text, '') = '' OR predicted_status = @predicted_status::text)
  AND e.deleted_at IS NULL
ORDER BY e.scheduled_at;

-- name: GetExecutionsByStatuses :many
SELECT
    sqlc.embed(e),
    sqlc.embed(r)
FROM
    test_workflow_executions e
        JOIN test_workflow_results r ON e.id = r.execution_id
WHERE r.status = ANY(@statuses::text[])
  AND e.deleted_at IS NULL
ORDER BY e.scheduled_at;
