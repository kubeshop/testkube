-- name: TransitionExecutionStatusAt :exec
UPDATE test_workflow_executions
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status = ANY(@from_statuses::text[]);

-- name: TransitionExecutionResultStatus :exec
UPDATE test_workflow_results
SET status = @to_status::text,
    predicted_status = COALESCE(@predicted_status::text, @to_status::text),
    finished_at = @finished_at
WHERE execution_id = @execution_id
  AND status = ANY(@from_statuses::text[]);

-- name: ForceCancelExecution :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status IN (
    'queued', 'assigned', 'starting', 'scheduling', 'running', 
    'pausing', 'paused', 'resuming', 'stopping'
  );

-- name: ForceCancelExecutionResult :exec
UPDATE test_workflow_results 
SET 
    status = 'canceled',
    predicted_status = 'canceled',
    finished_at = @finished_at
WHERE execution_id = @execution_id
  AND status IN (
    'queued', 'assigned', 'starting', 'scheduling', 'running', 
    'pausing', 'paused', 'resuming', 'stopping'
  );

-- name: ForceCancelExecutionSteps :exec
UPDATE test_workflow_results 
SET steps = (
    SELECT jsonb_object_agg(
        key,
        CASE 
            WHEN value->>'status' IN ('passed', 'failed') THEN value
            ELSE jsonb_build_object(
                'status', 'canceled',
                'queuedAt', COALESCE(
                    NULLIF(value->>'queuedAt', '0001-01-01T00:00:00Z'),
                    to_jsonb(@finished_at::timestamptz)
                ),
                'startedAt', COALESCE(
                    NULLIF(value->>'startedAt', '0001-01-01T00:00:00Z'),
                    to_jsonb(@finished_at::timestamptz)
                ),
                'finishedAt', COALESCE(
                    NULLIF(value->>'finishedAt', '0001-01-01T00:00:00Z'),
                    to_jsonb(@finished_at::timestamptz)
                )
            ) || (value - ARRAY['status', 'queuedat', 'startedat', 'finishedat'])
        END
    )
    FROM jsonb_each(COALESCE(steps, '{}'::jsonb))
)
WHERE execution_id = @execution_id
  AND steps IS NOT NULL
  AND jsonb_typeof(steps) = 'object';

-- name: ForceCancelExecutionInitialization :exec
UPDATE test_workflow_results 
SET initialization = (
    CASE 
        WHEN initialization->>'status' IN ('passed', 'failed') THEN initialization
        ELSE jsonb_build_object(
            'status', 'canceled',
            'queuedAt', COALESCE(
                NULLIF(initialization->>'queuedAt', '0001-01-01T00:00:00Z'),
                to_jsonb(@finished_at::timestamptz)
            ),
            'startedAt', COALESCE(
                NULLIF(initialization->>'startedAt', '0001-01-01T00:00:00Z'),
                to_jsonb(@finished_at::timestamptz)
            ),
            'finishedaAt', COALESCE(
                NULLIF(initialization->>'finishedAt', '0001-01-01T00:00:00Z'),
                to_jsonb(@finished_at::timestamptz)
            )
        ) || (initialization - ARRAY['status', 'queuedat', 'startedat', 'finishedat'])
    END
)
WHERE execution_id = @execution_id
  AND initialization IS NOT NULL
  AND jsonb_typeof(initialization) = 'object'
  AND initialization->>'status' NOT IN ('passed', 'failed');

-- name: GetExecutionForceCancel :one
SELECT 
    e.id,
    r.status
FROM test_workflow_executions e
INNER JOIN test_workflow_results r ON e.id = r.execution_id
WHERE e.id = @execution_id
  AND r.status IN (
    'queued', 'assigned', 'starting', 'scheduling', 'running', 
    'pausing', 'paused', 'resuming', 'stopping'
  );
