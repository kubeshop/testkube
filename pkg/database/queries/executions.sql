-- name: GetTestWorkflowExecution :one
SELECT * FROM test_workflow_executions 
WHERE id = @id OR name = @id;

-- name: GetTestWorkflowExecutionByNameAndTestWorkflow :one
SELECT * FROM test_workflow_executions 
WHERE (id = @name OR name = @name) AND workflow->>'name' = @workflow_name;

-- name: GetLatestTestWorkflowExecutionByTestWorkflow :one
SELECT * FROM test_workflow_executions 
WHERE workflow->>'name' = @workflow_name 
ORDER BY status_at DESC 
LIMIT 1;

-- name: GetLatestTestWorkflowExecutionsByTestWorkflows :many
SELECT DISTINCT ON (workflow->>'name') *
FROM test_workflow_executions 
WHERE workflow->>'name' = ANY(@workflow_names)
ORDER BY workflow->>'name', status_at DESC;

-- name: GetRunningTestWorkflowExecutions :many
SELECT * FROM test_workflow_executions 
WHERE result->>'status' IN ('paused', 'running', 'queued')
ORDER BY id DESC;

-- name: GetTestWorkflowExecutionsTotals :many
SELECT 
    result->>'status' as status,
    COUNT(*) as count
FROM test_workflow_executions 
WHERE 1=1
    AND (@workflow_name IS NULL OR workflow->>'name' = @workflow_name)
    AND (@workflow_names IS NULL OR workflow->>'name' = ANY(@workflow_names))
    AND (@text_search IS NULL OR name ILIKE '%' || @text_search || '%')
    AND (@start_date IS NULL OR scheduled_at >= @start_date)
    AND (@end_date IS NULL OR scheduled_at <= @end_date)
    AND (@last_n_days IS NULL OR scheduled_at >= NOW() - INTERVAL '@last_n_days days')
    AND (@statuses IS NULL OR result->>'status' = ANY(@statuses))
    AND (@runner_id IS NULL OR runner_id = @runner_id)
    AND (@assigned IS NULL OR 
         (@assigned = true AND runner_id IS NOT NULL AND runner_id != '') OR 
         (@assigned = false AND (runner_id IS NULL OR runner_id = '')))
    AND (@actor_name IS NULL OR running_context->'actor'->>'name' = @actor_name)
    AND (@actor_type IS NULL OR running_context->'actor'->>'type_' = @actor_type)
    AND (@group_id IS NULL OR id = @group_id OR group_id = @group_id)
    AND (@initialized IS NULL OR 
         (@initialized = true AND (result->>'status' != 'queued' OR result->'steps' IS NOT NULL)) OR
         (@initialized = false AND result->>'status' = 'queued' AND (result->'steps' IS NULL OR result->'steps' = '{}'::jsonb)))
GROUP BY result->>'status';

-- name: GetTestWorkflowExecutions :many
SELECT * FROM test_workflow_executions 
WHERE 1=1
    AND (@workflow_name IS NULL OR workflow->>'name' = @workflow_name)
    AND (@workflow_names IS NULL OR workflow->>'name' = ANY(@workflow_names))
    AND (@text_search IS NULL OR name ILIKE '%' || @text_search || '%')
    AND (@start_date IS NULL OR scheduled_at >= @start_date)
    AND (@end_date IS NULL OR scheduled_at <= @end_date)
    AND (@last_n_days IS NULL OR scheduled_at >= NOW() - INTERVAL '@last_n_days days')
    AND (@statuses IS NULL OR result->>'status' = ANY(@statuses))
    AND (@runner_id IS NULL OR runner_id = @runner_id)
    AND (@assigned IS NULL OR 
         (@assigned = true AND runner_id IS NOT NULL AND runner_id != '') OR 
         (@assigned = false AND (runner_id IS NULL OR runner_id = '')))
    AND (@actor_name IS NULL OR running_context->'actor'->>'name' = @actor_name)
    AND (@actor_type IS NULL OR running_context->'actor'->>'type_' = @actor_type)
    AND (@group_id IS NULL OR id = @group_id OR group_id = @group_id)
    AND (@initialized IS NULL OR 
         (@initialized = true AND (result->>'status' != 'queued' OR result->'steps' IS NOT NULL)) OR
         (@initialized = false AND result->>'status' = 'queued' AND (result->'steps' IS NULL OR result->'steps' = '{}'::jsonb)))
ORDER BY scheduled_at DESC
LIMIT @lmt OFFSET @ofst;

-- name: GetTestWorkflowExecutionsSummary :many
SELECT 
    id, group_id, runner_id, name, number, scheduled_at, status_at,
    result, workflow, tags, running_context, config_params, reports, resource_aggregations
FROM test_workflow_executions 
WHERE 1=1
    AND (@workflow_name IS NULL OR workflow->>'name' = @workflow_name)
    AND (@workflow_names IS NULL OR workflow->>'name' = ANY(@workflow_names))
    AND (@text_search IS NULL OR name ILIKE '%' || @text_search || '%')
    AND (@start_date IS NULL OR scheduled_at >= @start_date)
    AND (@end_date IS NULL OR scheduled_at <= @end_date)
    AND (@last_n_days IS NULL OR scheduled_at >= NOW() - INTERVAL '@last_n_days days')
    AND (@statuses IS NULL OR result->>'status' = ANY(@statuses))
    AND (@runner_id IS NULL OR runner_id = @runner_id)
    AND (@assigned IS NULL OR 
         (@assigned = true AND runner_id IS NOT NULL AND runner_id != '') OR 
         (@assigned = false AND (runner_id IS NULL OR runner_id = '')))
    AND (@actor_name IS NULL OR running_context->'actor'->>'name' = @actor_name)
    AND (@actor_type IS NULL OR running_context->'actor'->>'type_' = @actor_type)
    AND (@group_id IS NULL OR id = @group_id OR group_id = @group_id)
    AND (@initialized IS NULL OR 
         (@initialized = true AND (result->>'status' != 'queued' OR result->'steps' IS NOT NULL)) OR
         (@initialized = false AND result->>'status' = 'queued' AND (result->'steps' IS NULL OR result->'steps' = '{}'::jsonb)))
ORDER BY scheduled_at DESC
LIMIT @lmt OFFSET @ofst;

-- name: InsertTestWorkflowExecution :exec
INSERT INTO test_workflow_executions (
    id, group_id, runner_id, runner_target, runner_original_target, name, namespace, number,
    scheduled_at, assigned_at, status_at, signature, result, output, reports, resource_aggregations,
    workflow, resolved_workflow, test_workflow_execution_name, disable_webhooks, tags, 
    running_context, config_params
) VALUES (
    @id, @group_id, @runner_id, @runner_target, @runner_original_target, @name, @namespace, @number,
    @scheduled_at, @assigned_at, @status_at, @signature, @result, @output, @reports, @resource_aggregations,
    @workflow, @resolved_workflow, @test_workflow_execution_name, @disable_webhooks, @tags,
    @running_context, @config_params
);

-- name: UpdateTestWorkflowExecution :exec
UPDATE test_workflow_executions 
SET 
    group_id = @group_id,
    runner_id = @runner_id,
    runner_target = @runner_target,
    runner_original_target = @runner_original_target,
    name = @name,
    namespace = @namespace,
    number = @number,
    scheduled_at = @scheduled_at,
    assigned_at = @assigned_at,
    status_at = @status_at,
    signature = @signature,
    result = @result,
    output = @output,
    reports = @reports,
    resource_aggregations = @resource_aggregations,
    workflow = @workflow,
    resolved_workflow = @resolved_workflow,
    test_workflow_execution_name = @test_workflow_execution_name,
    disable_webhooks = @disable_webhooks,
    tags = @tags,
    running_context = @running_context,
    config_params = @config_params
WHERE id = @id;

-- name: UpdateTestWorkflowExecutionResult :exec
UPDATE test_workflow_executions 
SET 
    result = @result,
    status_at = CASE 
        WHEN @finished_at IS NOT NULL THEN @finished_at 
        ELSE status_at 
    END
WHERE id = @id;

-- name: UpdateTestWorkflowExecutionReport :exec
UPDATE test_workflow_executions 
SET reports = COALESCE(reports, '[]'::jsonb) || @report::jsonb
WHERE id = @id;

-- name: UpdateTestWorkflowExecutionOutput :exec
UPDATE test_workflow_executions 
SET output = @output
WHERE id = @id;

-- name: UpdateTestWorkflowExecutionResourceAggregations :exec
UPDATE test_workflow_executions 
SET resource_aggregations = @resource_aggregations
WHERE id = @id;

-- name: DeleteTestWorkflowExecutionsByTestWorkflow :exec
DELETE FROM test_workflow_executions 
WHERE workflow->>'name' = @workflow_name;

-- name: DeleteAllTestWorkflowExecutions :exec
DELETE FROM test_workflow_executions;

-- name: DeleteTestWorkflowExecutionsByTestWorkflows :exec
DELETE FROM test_workflow_executions 
WHERE workflow->>'name' = ANY(@workflow_names);

-- name: GetTestWorkflowMetrics :many
SELECT 
    id as execution_id,
    group_id,
    result->>'duration' as duration,
    result->>'durationms' as duration_ms,
    result->>'status' as status,
    name,
    scheduled_at as start_time,
    runner_id
FROM test_workflow_executions 
WHERE workflow->>'name' = @workflow_name
    AND (@last_days = 0 OR scheduled_at >= NOW() - INTERVAL '@last_days days')
ORDER BY scheduled_at DESC
LIMIT @lmt;

-- name: GetPreviousFinishedState :one
SELECT result->>'status' as status
FROM test_workflow_executions 
WHERE workflow->>'name' = @workflow_name
    AND (result->>'finishedAt')::timestamp < @date
    AND result->>'status' IN ('passed', 'failed', 'skipped', 'aborted', 'timeout')
ORDER BY (result->>'finishedAt')::timestamp DESC
LIMIT 1;

-- name: InitTestWorkflowExecution :exec
UPDATE test_workflow_executions 
SET 
    namespace = @namespace,
    signature = @signature,
    runner_id = @runner_id
WHERE id = @id;

-- name: AssignTestWorkflowExecution :one
UPDATE test_workflow_executions 
SET 
    runner_id = @new_runner_id,
    assigned_at = @assigned_at
WHERE id = @id
    AND result->>'status' = 'queued'
    AND (runner_id = @prev_runner_id OR runner_id = @new_runner_id OR runner_id IS NULL)
RETURNING id;

-- name: GetUnassignedTestWorkflowExecutions :many
SELECT * FROM test_workflow_executions 
WHERE result->>'status' = 'queued'
    AND (runner_id IS NULL OR runner_id = '')
ORDER BY id DESC;

-- name: AbortTestWorkflowExecutionIfQueued :one
UPDATE test_workflow_executions 
SET 
    result = jsonb_set(
        jsonb_set(
            jsonb_set(
                jsonb_set(
                    jsonb_set(
                        jsonb_set(result, '{status}', '"aborted"'),
                        '{predictedstatus}', '"aborted"'
                    ),
                    '{finishedat}', to_jsonb(@abort_time::timestamp)
                ),
                '{initialization,status}', '"aborted"'
            ),
            '{initialization,errormessage}', '"Aborted before initialization."'
        ),
        '{initialization,finishedat}', to_jsonb(@abort_time::timestamp)
    ),
    status_at = @abort_time
WHERE id = @id
    AND result->>'status' IN ('queued', 'running', 'paused')
    AND (runner_id IS NULL OR runner_id = '')
RETURNING id;

-- name: GetNextExecutionNumber :one
SELECT nextval('test_workflow_execution_number_seq_' || @workflow_name) as number;
