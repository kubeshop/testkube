-- name: StartExecution :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status = 'assigned';

-- name: StartExecutionResult :exec
UPDATE test_workflow_results 
SET status = 'starting'
WHERE execution_id = @execution_id
  AND status = 'assigned';

-- name: PauseExecution :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status = 'running';

-- name: PauseExecutionResult :exec
UPDATE test_workflow_results 
SET status = 'pausing'
WHERE execution_id = @execution_id
  AND status = 'running';

-- name: ResumeExecution :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status = 'paused';

-- name: ResumeExecutionResult :exec
UPDATE test_workflow_results 
SET status = 'resuming'
WHERE execution_id = @execution_id
  AND status = 'paused';

-- name: AbortExecutionRunning :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status IN ('starting', 'scheduling', 'running', 'paused', 'resuming');

-- name: AbortExecutionRunningResult :exec
UPDATE test_workflow_results 
SET 
    status = 'stopping',
    predicted_status = 'aborted'
WHERE execution_id = @execution_id
  AND status IN ('starting', 'scheduling', 'running', 'paused', 'resuming');

-- name: AbortExecutionQueued :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status IN ('queued', 'assigned');

-- name: AbortExecutionQueuedResult :exec
UPDATE test_workflow_results 
SET 
    status = 'aborted',
    predicted_status = 'aborted',
    finished_at = @finished_at
WHERE execution_id = @execution_id
  AND status IN ('queued', 'assigned');

-- name: CancelExecutionRunning :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status IN ('starting', 'scheduling', 'running', 'paused', 'resuming');

-- name: CancelExecutionRunningResult :exec
UPDATE test_workflow_results 
SET 
    status = 'stopping',
    predicted_status = 'canceled'
WHERE execution_id = @execution_id
  AND status IN ('starting', 'scheduling', 'running', 'paused', 'resuming');

-- name: CancelExecutionQueued :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @execution_id
  AND test_workflow_executions.id = r.execution_id
  AND r.status IN ('queued', 'assigned');

-- name: CancelExecutionQueuedResult :exec
UPDATE test_workflow_results 
SET 
    status = 'canceled',
    predicted_status = 'canceled',
    finished_at = @finished_at
WHERE execution_id = @execution_id
  AND status IN ('queued', 'assigned');

-- name: ForceCancelExecution :exec
-- Updates the execution status_at timestamp for executions in cancellable states
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
-- Updates the main result status and timestamps
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
-- Cancels all steps that are not already terminated (passed/failed)
-- Sets missing timestamps (queuedat, startedat, finishedat) to the provided time
UPDATE test_workflow_results 
SET steps = (
    SELECT jsonb_object_agg(
        key,
        CASE 
            -- If step is already passed or failed, keep it as is
            WHEN value->>'status' IN ('passed', 'failed') THEN value
            -- Otherwise, cancel it and update timestamps
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
-- Cancels the initialization step if it's not already terminated
UPDATE test_workflow_results 
SET initialization = (
    CASE 
        -- If initialization is already passed or failed, keep it as is
        WHEN initialization->>'status' IN ('passed', 'failed') THEN initialization
        -- Otherwise, cancel it and update timestamps
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
-- Helper query to verify the execution can be force-canceled
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
