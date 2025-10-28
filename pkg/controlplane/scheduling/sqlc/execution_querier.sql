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
