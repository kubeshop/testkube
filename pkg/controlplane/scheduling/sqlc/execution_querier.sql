-- name: GetExecutionsByStatus :many
SELECT
    sqlc.embed(e),
    sqlc.embed(r)
FROM
    test_workflow_executions e
        JOIN test_workflow_results r ON e.id = r.execution_id
WHERE r.status = @status::text
  AND (COALESCE(@predicted_status::text, '') = '' OR predicted_status = @predicted_status::text)
ORDER BY e.scheduled_at;

-- name: GetExecutionsByStatuses :many
SELECT
    sqlc.embed(e),
    sqlc.embed(r)
FROM
    test_workflow_executions e
        JOIN test_workflow_results r ON e.id = r.execution_id
WHERE e.status = ANY(@statuses::text[])
  AND (
    COALESCE(e.status_at, e.scheduled_at) <= @snapshot_before::timestamptz
    )
  AND (
    @after_pending_at::timestamptz IS NULL
        OR COALESCE(e.status_at, e.scheduled_at) > @after_pending_at::timestamptz
        OR (COALESCE(e.status_at, e.scheduled_at) = @after_pending_at::timestamptz AND e.id > @after_execution_id::text)
    )
ORDER BY COALESCE(e.status_at, e.scheduled_at), e.id
LIMIT @row_limit::int;
