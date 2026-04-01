-- name: GetTestWorkflowExecution :one
SELECT * FROM v_test_workflow_execution_details
WHERE (id = @id OR name = @id) AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: GetTestWorkflowExecutionByNameAndTestWorkflow :one
SELECT * FROM v_test_workflow_execution_details
WHERE (id = @name OR name = @name) AND workflow_name = @workflow_name::text AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: GetLatestTestWorkflowExecutionByTestWorkflow :one
SELECT * FROM v_test_workflow_execution_details
WHERE workflow_name = @workflow_name::text AND (organization_id = @organization_id AND environment_id = @environment_id)
ORDER BY
    CASE
        WHEN @sort_by_number::boolean = true AND @sort_by_status::boolean = false THEN number
        WHEN @sort_by_status::boolean = true AND @sort_by_number::boolean = false THEN EXTRACT(EPOCH FROM status_at)::integer
    ELSE
        EXTRACT(EPOCH FROM scheduled_at)::integer
    END DESC
LIMIT 1;

-- name: GetLatestTestWorkflowExecutionsByTestWorkflows :many
SELECT DISTINCT ON (workflow_name) * FROM v_test_workflow_execution_details
WHERE workflow_name = ANY(@workflow_names::text[]) AND (organization_id = @organization_id AND environment_id = @environment_id)
ORDER BY workflow_name, status_at DESC;

-- name: GetRunningTestWorkflowExecutions :many
SELECT * FROM v_test_workflow_execution_details
WHERE status IN ('queued', 'assigned', 'starting', 'running', 'pausing', 'paused', 'resuming') AND (organization_id = @organization_id AND environment_id = @environment_id)
ORDER BY id DESC;

-- name: GetFinishedTestWorkflowExecutions :many
SELECT * FROM v_test_workflow_execution_details
WHERE status IN ('passed', 'failed', 'aborted') AND (organization_id = @organization_id AND environment_id = @environment_id)
    AND (silent_mode IS NULL OR (silent_mode->>'health')::boolean IS NOT TRUE)
    AND (COALESCE(@workflow_name::text, '') = '' OR workflow_name = @workflow_name::text)
    AND (COALESCE(@workflow_names::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR workflow_name = ANY(@workflow_names::text[]))
    AND (COALESCE(@text_search::text, '') = '' OR name ILIKE '%' || @text_search::text || '%')
    AND (COALESCE(@start_date::timestamptz, '1900-01-01'::timestamptz) = '1900-01-01'::timestamptz OR scheduled_at >= @start_date::timestamptz)
    AND (COALESCE(@end_date::timestamptz, '2100-01-01'::timestamptz) = '2100-01-01'::timestamptz OR scheduled_at <= @end_date::timestamptz)
    AND (COALESCE(@last_n_days::integer, 0) = 0 OR scheduled_at >= NOW() - (COALESCE(@last_n_days::integer, 0) || ' days')::interval)
    AND (COALESCE(@statuses::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR status = ANY(@statuses::text[]))
    AND (COALESCE(@runner_id::text, '') = '' OR runner_id = @runner_id::text)
    AND (COALESCE(@assigned, NULL) IS NULL OR
         (@assigned::boolean = true AND runner_id IS NOT NULL AND runner_id != '') OR
         (@assigned::boolean = false AND (runner_id IS NULL OR runner_id = '')))
    AND (COALESCE(@actor_name::text, '') = '' OR running_context->'actor'->>'name' = @actor_name::text)
    AND (COALESCE(@actor_type::text, '') = '' OR running_context->'actor'->>'type_' = @actor_type::text)
    AND (COALESCE(@group_id::text, '') = '' OR id = @group_id::text OR group_id = @group_id::text)
    AND (COALESCE(@initialized, NULL) IS NULL OR
         (@initialized::boolean = true AND (status != 'queued' OR steps IS NOT NULL)) OR
         (@initialized::boolean = false AND status = 'queued' AND (steps IS NULL OR steps = '{}'::jsonb)))
   AND (COALESCE(@health_ranges::jsonb, '[]'::jsonb) = '[]'::jsonb OR
          EXISTS (
              SELECT 1 FROM jsonb_array_elements(@health_ranges::jsonb) AS range_obj
              WHERE (workflow_status->>'health')::jsonb->>'overallHealth' IS NOT NULL
                AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision >= (range_obj->>'min')::double precision
                AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision <= (range_obj->>'max')::double precision
          )
      )
    AND (
        (COALESCE(@tag_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (tags ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        tags ? key_condition
                END
            ) = array_length(@tag_keys::text[], 1)
        )
        AND
        (COALESCE(@tag_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_conditions::text[]) AS cond
                WHERE tags->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@label_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) > 0
        )
        AND
        (COALESCE(@label_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@selector_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@selector_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) = array_length(@selector_keys::text[], 1)
        )
        AND
        (COALESCE(@selector_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) = (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond)
        )
    )
ORDER BY scheduled_at DESC
LIMIT NULLIF(@lmt, 0) OFFSET @fst;

-- name: GetTestWorkflowExecutionsTotals :many
SELECT
    status,
    COUNT(*) as count
FROM v_test_workflow_execution_details
WHERE (organization_id = @organization_id AND environment_id = @environment_id)
    AND (COALESCE(@workflow_name::text, '') = '' OR workflow_name = @workflow_name::text)
    AND (COALESCE(@workflow_names::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR workflow_name = ANY(@workflow_names::text[]))
    AND (COALESCE(@text_search::text, '') = '' OR name ILIKE '%' || @text_search::text || '%')
    AND (COALESCE(@start_date::timestamptz, '1900-01-01'::timestamptz) = '1900-01-01'::timestamptz OR scheduled_at >= @start_date::timestamptz)
    AND (COALESCE(@end_date::timestamptz, '2100-01-01'::timestamptz) = '2100-01-01'::timestamptz OR scheduled_at <= @end_date::timestamptz)
    AND (COALESCE(@last_n_days::integer, 0) = 0 OR scheduled_at >= NOW() - (COALESCE(@last_n_days::integer, 0) || ' days')::interval)
    AND (COALESCE(@statuses::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR status = ANY(@statuses::text[]))
    AND (COALESCE(@runner_id::text, '') = '' OR runner_id = @runner_id::text)
    AND (COALESCE(@assigned, NULL) IS NULL OR
         (@assigned::boolean = true AND runner_id IS NOT NULL AND runner_id != '') OR
         (@assigned::boolean = false AND (runner_id IS NULL OR runner_id = '')))
    AND (COALESCE(@actor_name::text, '') = '' OR running_context->'actor'->>'name' = @actor_name::text)
    AND (COALESCE(@actor_type::text, '') = '' OR running_context->'actor'->>'type_' = @actor_type::text)
    AND (COALESCE(@group_id::text, '') = '' OR id = @group_id::text OR group_id = @group_id::text)
    AND (COALESCE(@initialized, NULL) IS NULL OR
         (@initialized::boolean = true AND (status != 'queued' OR steps IS NOT NULL)) OR
         (@initialized::boolean = false AND status = 'queued' AND (steps IS NULL OR steps = '{}'::jsonb)))
    AND (COALESCE(@health_ranges::jsonb, '[]'::jsonb) = '[]'::jsonb OR
          EXISTS (
              SELECT 1 FROM jsonb_array_elements(@health_ranges::jsonb) AS range_obj
              WHERE (workflow_status->>'health')::jsonb->>'overallHealth' IS NOT NULL
                AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision >= (range_obj->>'min')::double precision
                AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision <= (range_obj->>'max')::double precision
          )
      )
    AND (
        (COALESCE(@tag_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (tags ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        tags ? key_condition
                END
            ) = array_length(@tag_keys::text[], 1)
        )
        AND
        (COALESCE(@tag_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_conditions::text[]) AS cond
                WHERE tags->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@label_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) > 0
        )
        AND
        (COALESCE(@label_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@selector_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@selector_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) = array_length(@selector_keys::text[], 1)
        )
        AND
        (COALESCE(@selector_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) = (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond)
        )
    )
GROUP BY status;

-- name: GetTestWorkflowExecutions :many
SELECT * FROM v_test_workflow_execution_details
WHERE (organization_id = @organization_id AND environment_id = @environment_id)
    AND (COALESCE(@workflow_name::text, '') = '' OR workflow_name = @workflow_name::text)
    AND (COALESCE(@workflow_names::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR workflow_name = ANY(@workflow_names::text[]))
    AND (COALESCE(@text_search::text, '') = '' OR name ILIKE '%' || @text_search::text || '%')
    AND (COALESCE(@start_date::timestamptz, '1900-01-01'::timestamptz) = '1900-01-01'::timestamptz OR scheduled_at >= @start_date::timestamptz)
    AND (COALESCE(@end_date::timestamptz, '2100-01-01'::timestamptz) = '2100-01-01'::timestamptz OR scheduled_at <= @end_date::timestamptz)
    AND (COALESCE(@last_n_days::integer, 0) = 0 OR scheduled_at >= NOW() - (COALESCE(@last_n_days::integer, 0) || ' days')::interval)
    AND (COALESCE(@statuses::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR status = ANY(@statuses::text[]))
    AND (COALESCE(@runner_id::text, '') = '' OR runner_id = @runner_id::text)
    AND (COALESCE(@assigned, NULL) IS NULL OR
         (@assigned::boolean = true AND runner_id IS NOT NULL AND runner_id != '') OR
         (@assigned::boolean = false AND (runner_id IS NULL OR runner_id = '')))
    AND (COALESCE(@actor_name::text, '') = '' OR running_context->'actor'->>'name' = @actor_name::text)
    AND (COALESCE(@actor_type::text, '') = '' OR running_context->'actor'->>'type_' = @actor_type::text)
    AND (COALESCE(@group_id::text, '') = '' OR id = @group_id::text OR group_id = @group_id::text)
    AND (COALESCE(@initialized, NULL) IS NULL OR
         (@initialized::boolean = true AND (status != 'queued' OR steps IS NOT NULL)) OR
         (@initialized::boolean = false AND status = 'queued' AND (steps IS NULL OR steps = '{}'::jsonb)))
   AND (COALESCE(@health_ranges::jsonb, '[]'::jsonb) = '[]'::jsonb OR
          EXISTS (
              SELECT 1 FROM jsonb_array_elements(@health_ranges::jsonb) AS range_obj
              WHERE (workflow_status->>'health')::jsonb->>'overallHealth' IS NOT NULL
                AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision >= (range_obj->>'min')::double precision
                AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision <= (range_obj->>'max')::double precision
          )
      )
    AND (
        (COALESCE(@tag_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (tags ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        tags ? key_condition
                END
            ) = array_length(@tag_keys::text[], 1)
        )
        AND
        (COALESCE(@tag_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_conditions::text[]) AS cond
                WHERE tags->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@label_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) > 0
        )
        AND
        (COALESCE(@label_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@selector_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@selector_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) = array_length(@selector_keys::text[], 1)
        )
        AND
        (COALESCE(@selector_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) = (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond)
        )
    )
ORDER BY scheduled_at DESC
LIMIT NULLIF(@lmt, 0) OFFSET @fst;

-- name: InsertTestWorkflowExecution :exec
INSERT INTO test_workflow_executions (
    id, group_id, runner_id, runner_target, runner_original_target, name, namespace, number,
    scheduled_at, assigned_at, status_at, test_workflow_execution_name, disable_webhooks,
    tags, running_context, config_params, organization_id, environment_id, runtime, silent_mode
) VALUES (
    @id, @group_id, @runner_id, @runner_target, @runner_original_target, @name, @namespace, @number,
    @scheduled_at, @assigned_at, @status_at, @test_workflow_execution_name, @disable_webhooks,
    @tags, @running_context, @config_params, @organization_id, @environment_id, @runtime, @silent_mode
);

-- name: InsertTestWorkflowSignature :one
INSERT INTO test_workflow_signatures (
    execution_id, ref, name, category, optional, negative, parent_id, sig_order
) VALUES (
    @execution_id, @ref, @name, @category, @optional, @negative, @parent_id, @sig_order
)
RETURNING test_workflow_signatures.id;

-- name: InsertTestWorkflowResult :exec
INSERT INTO test_workflow_results (
    execution_id, status, predicted_status, queued_at, started_at, finished_at,
    duration, total_duration, duration_ms, paused_ms, total_duration_ms,
    pauses, initialization, steps
) VALUES (
    @execution_id, @status, @predicted_status, @queued_at, @started_at, @finished_at,
    @duration, @total_duration, @duration_ms, @paused_ms, @total_duration_ms,
    @pauses, @initialization, @steps
)
ON CONFLICT (execution_id) DO UPDATE SET
    status = EXCLUDED.status,
    predicted_status = EXCLUDED.predicted_status,
    queued_at = EXCLUDED.queued_at,
    started_at = EXCLUDED.started_at,
    finished_at = EXCLUDED.finished_at,
    duration = EXCLUDED.duration,
    total_duration = EXCLUDED.total_duration,
    duration_ms = EXCLUDED.duration_ms,
    paused_ms = EXCLUDED.paused_ms,
    total_duration_ms = EXCLUDED.total_duration_ms,
    pauses = EXCLUDED.pauses,
    initialization = EXCLUDED.initialization,
    steps = EXCLUDED.steps;

-- name: InsertTestWorkflowOutput :exec
INSERT INTO test_workflow_outputs (execution_id, ref, name, value, out_order)
VALUES (@execution_id, @ref, @name, @value, @out_order);

-- name: InsertTestWorkflowReport :exec
INSERT INTO test_workflow_reports (execution_id, ref, kind, file, summary, rep_order)
VALUES (@execution_id, @ref, @kind, @file, @summary, @rep_order);

-- name: InsertTestWorkflowResourceAggregations :exec
INSERT INTO test_workflow_resource_aggregations (execution_id, global, step)
VALUES (@execution_id, @global, @step)
ON CONFLICT (execution_id) DO UPDATE SET
    global = EXCLUDED.global,
    step = EXCLUDED.step;

-- name: InsertTestWorkflow :exec
INSERT INTO test_workflows (
    execution_id, workflow_type, name, namespace, description, labels, annotations,
    created, updated, spec, read_only, status
) VALUES (
    @execution_id, @workflow_type, @name, @namespace, @description, @labels, @annotations,
    @created, @updated, @spec, @read_only, @status
)
ON CONFLICT (execution_id, workflow_type) DO UPDATE SET
    name = EXCLUDED.name,
    namespace = EXCLUDED.namespace,
    description = EXCLUDED.description,
    labels = EXCLUDED.labels,
    annotations = EXCLUDED.annotations,
    created = EXCLUDED.created,
    updated = EXCLUDED.updated,
    spec = EXCLUDED.spec,
    read_only = EXCLUDED.read_only,
    status = EXCLUDED.status;

-- name: UpdateTestWorkflowExecutionResult :exec
UPDATE test_workflow_results 
SET 
    status = @status,
    predicted_status = @predicted_status,
    queued_at = @queued_at,
    started_at = @started_at,
    finished_at = @finished_at,
    duration = @duration,
    total_duration = @total_duration,
    duration_ms = @duration_ms,
    paused_ms = @paused_ms,
    total_duration_ms = @total_duration_ms,
    pauses = @pauses,
    initialization = @initialization,
    steps = @steps
WHERE execution_id = @execution_id;

-- name: UpdateExecutionStatus :exec
UPDATE test_workflow_results 
SET 
    status = @status
WHERE execution_id = @execution_id;

-- name: UpdateExecutionStatusAt :exec
UPDATE test_workflow_executions 
SET status_at = @status_at
WHERE id = @execution_id AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: DeleteTestWorkflowSignatures :exec
DELETE FROM test_workflow_signatures WHERE execution_id = @execution_id;

-- name: DeleteTestWorkflowResult :exec
DELETE FROM test_workflow_results WHERE execution_id = @execution_id;

-- name: DeleteTestWorkflowOutputs :exec
DELETE FROM test_workflow_outputs WHERE execution_id = @execution_id;

-- name: DeleteTestWorkflowReports :exec
DELETE FROM test_workflow_reports WHERE execution_id = @execution_id;

-- name: DeleteTestWorkflowResourceAggregations :exec
DELETE FROM test_workflow_resource_aggregations WHERE execution_id = @execution_id;

-- name: DeleteTestWorkflow :exec
DELETE FROM test_workflows WHERE execution_id = @execution_id AND workflow_type = @workflow_type;

-- name: UpdateTestWorkflowExecutionResourceAggregations :exec
UPDATE test_workflow_resource_aggregations
SET
    global = @global,
    step = @step
WHERE execution_id = @execution_id;

-- name: UpdateTestWorkflowExecutionTags :execrows
UPDATE test_workflow_executions
SET tags = @tags
WHERE (id = @execution_id OR name = @execution_id) AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: DeleteTestWorkflowExecutionsByTestWorkflow :exec
DELETE FROM test_workflow_executions e
USING test_workflows w
WHERE e.id = w.execution_id AND (e.organization_id = @organization_id AND e.environment_id = @environment_id)
  AND w.workflow_type = 'workflow' 
  AND w.name = @workflow_name::text;

-- name: DeleteAllTestWorkflowExecutions :exec
DELETE FROM test_workflow_executions WHERE organization_id = @organization_id AND environment_id = @environment_id;

-- name: DeleteTestWorkflowExecutionsByTestWorkflows :exec
DELETE FROM test_workflow_executions e
USING test_workflows w
WHERE e.id = w.execution_id AND (e.organization_id = @organization_id AND e.environment_id = @environment_id)
  AND w.workflow_type = 'workflow' 
  AND w.name = ANY(@workflow_names::text[]);

-- name: GetTestWorkflowMetrics :many
SELECT 
    e.id as execution_id,
    e.group_id,
    r.duration,
    r.duration_ms,
    r.status,
    e.name,
    e.scheduled_at as start_time,
    e.runner_id
FROM test_workflow_executions e
LEFT JOIN test_workflow_results r ON e.id = r.execution_id
LEFT JOIN test_workflows w ON e.id = w.execution_id AND w.workflow_type = 'workflow'
WHERE w.name = @workflow_name::text AND (e.organization_id = @organization_id AND e.environment_id = @environment_id)
    AND (@last_n_days::integer = 0 OR e.scheduled_at >= NOW() - (@last_n_days::integer || ' days')::interval)
ORDER BY e.scheduled_at DESC
LIMIT NULLIF(@lmt, 0);

-- name: GetPreviousFinishedState :one
SELECT r.status
FROM test_workflow_executions e
LEFT JOIN test_workflow_results r ON e.id = r.execution_id
LEFT JOIN test_workflows w ON e.id = w.execution_id AND w.workflow_type = 'workflow'
WHERE w.name = @workflow_name::text AND (e.organization_id = @organization_id AND e.environment_id = @environment_id)
    AND r.finished_at < @date
    AND r.status IN ('passed', 'failed', 'skipped', 'aborted', 'canceled', 'timeout')
ORDER BY r.finished_at DESC
LIMIT 1;

-- name: GetTestWorkflowExecutionTags :many
WITH tag_extracts AS (
    SELECT 
        e.id,
        w.name as workflow_name,
        tag_pair.key as tag_key,
        tag_pair.value as tag_value
    FROM test_workflow_executions e
    LEFT JOIN test_workflows w ON e.id = w.execution_id AND w.workflow_type = 'workflow'
    CROSS JOIN LATERAL jsonb_each_text(e.tags) AS tag_pair(key, value)
    WHERE e.tags IS NOT NULL AND (e.organization_id = @organization_id AND e.environment_id = @environment_id)
        AND e.tags != '{}'::jsonb
        AND jsonb_typeof(e.tags) = 'object'
)
SELECT 
    tag_key::text,
    array_agg(DISTINCT tag_value ORDER BY tag_value)::text[] as values
FROM tag_extracts
WHERE (COALESCE(@workflow_name::text, '') = '' OR workflow_name = @workflow_name::text)
GROUP BY tag_key
ORDER BY tag_key;

-- name: InitTestWorkflowExecution :exec
UPDATE test_workflow_executions 
SET 
    namespace = @namespace,
    runner_id = @runner_id,
    status_at = NOW()
WHERE id = @id AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: AssignTestWorkflowExecution :one
UPDATE test_workflow_executions 
SET 
    runner_id = @new_runner_id::text,
    assigned_at = @assigned_at
FROM test_workflow_results r
WHERE test_workflow_executions.id = @id AND (test_workflow_executions.organization_id = @organization_id AND test_workflow_executions.environment_id = @environment_id)
    AND test_workflow_executions.id = r.execution_id
    AND r.status = 'queued'
    AND ((test_workflow_executions.runner_id IS NULL OR test_workflow_executions.runner_id = '')
         OR (test_workflow_executions.runner_id = @new_runner_id::text AND assigned_at < @assigned_at)
         OR (test_workflow_executions.runner_id = @prev_runner_id::text AND assigned_at < NOW() - INTERVAL '1 minute' AND assigned_at < @assigned_at))
RETURNING test_workflow_executions.id;

-- name: GetUnassignedTestWorkflowExecutions :many
SELECT * FROM v_test_workflow_execution_details
WHERE status = 'queued' AND (organization_id = @organization_id AND environment_id = @environment_id)
    AND (runner_id IS NULL OR runner_id = '')
ORDER BY id DESC;

-- name: AbortTestWorkflowExecutionIfQueued :one
UPDATE test_workflow_executions 
SET status_at = @abort_time
FROM test_workflow_results r
WHERE test_workflow_executions.id = @id AND (test_workflow_executions.organization_id = @organization_id AND test_workflow_executions.environment_id = @environment_id)
    AND test_workflow_executions.id = r.execution_id
    AND r.status IN ('queued', 'assigned', 'starting', 'running', 'paused', 'resuming')
    AND (test_workflow_executions.runner_id IS NULL OR test_workflow_executions.runner_id = '')
RETURNING test_workflow_executions.id;

-- name: AbortTestWorkflowResultIfQueued :exec
UPDATE test_workflow_results 
SET 
    status = 'aborted',
    predicted_status = 'aborted',
    finished_at = @abort_time,
    initialization = jsonb_set(
        jsonb_set(
            jsonb_set(COALESCE(initialization, '{}'::jsonb), '{status}', '"aborted"'),
            '{errormessage}', '"Aborted before initialization."'
        ),
        '{finishedat}', to_jsonb(@abort_time::timestamp)
    )
WHERE execution_id = @id
    AND status IN ('queued', 'running', 'paused');

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
    test_workflow_execution_name = @test_workflow_execution_name,
    disable_webhooks = @disable_webhooks,
    tags = @tags,
    running_context = @running_context,
    config_params = @config_params,
    runtime = @runtime,
    silent_mode = @silent_mode
WHERE id = @id AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: GetTestWorkflowExecutionsSummary :many
SELECT * FROM v_test_workflow_execution_details
WHERE (organization_id = @organization_id AND environment_id = @environment_id)
    AND (COALESCE(@workflow_name::text, '') = '' OR workflow_name = @workflow_name::text)
    AND (COALESCE(@workflow_names::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR workflow_name = ANY(@workflow_names::text[]))
    AND (COALESCE(@text_search::text, '') = '' OR name ILIKE '%' || @text_search::text || '%')
    AND (COALESCE(@start_date::timestamptz, '1900-01-01'::timestamptz) = '1900-01-01'::timestamptz OR scheduled_at >= @start_date::timestamptz)
    AND (COALESCE(@end_date::timestamptz, '2100-01-01'::timestamptz) = '2100-01-01'::timestamptz OR scheduled_at <= @end_date::timestamptz)
    AND (COALESCE(@last_n_days::integer, 0) = 0 OR scheduled_at >= NOW() - (COALESCE(@last_n_days::integer, 0) || ' days')::interval)
    AND (COALESCE(@statuses::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR status = ANY(@statuses::text[]))
    AND (COALESCE(@runner_id::text, '') = '' OR runner_id = @runner_id::text)
    AND (COALESCE(@assigned, NULL) IS NULL OR
         (@assigned::boolean = true AND runner_id IS NOT NULL AND runner_id != '') OR
         (@assigned::boolean = false AND (runner_id IS NULL OR runner_id = '')))
    AND (COALESCE(@actor_name::text, '') = '' OR running_context->'actor'->>'name' = @actor_name::text)
    AND (COALESCE(@actor_type::text, '') = '' OR running_context->'actor'->>'type_' = @actor_type::text)
    AND (COALESCE(@group_id::text, '') = '' OR id = @group_id::text OR group_id = @group_id::text)
    AND (COALESCE(@initialized, NULL) IS NULL OR
         (@initialized::boolean = true AND (status != 'queued' OR steps IS NOT NULL)) OR
         (@initialized::boolean = false AND status = 'queued' AND (steps IS NULL OR steps = '{}'::jsonb)))
    AND (COALESCE(@health_ranges::jsonb, '[]'::jsonb) = '[]'::jsonb OR
         EXISTS (
             SELECT 1 FROM jsonb_array_elements(@health_ranges::jsonb) AS range_obj
             WHERE (workflow_status->>'health')::jsonb->>'overallHealth' IS NOT NULL
               AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision >= (range_obj->>'min')::double precision
               AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision <= (range_obj->>'max')::double precision
         )
    )
    AND (
        (COALESCE(@tag_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (tags ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        tags ? key_condition
                END
            ) = array_length(@tag_keys::text[], 1)
        )
        AND
        (COALESCE(@tag_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@tag_conditions::text[]) AS cond
                WHERE tags->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@label_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) > 0
        )
        AND
        (COALESCE(@label_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@selector_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@selector_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) = array_length(@selector_keys::text[], 1)
        )
        AND
        (COALESCE(@selector_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) = (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond)
        )
    )
ORDER BY scheduled_at DESC
LIMIT NULLIF(@lmt, 0) OFFSET @fst;

-- name: CountTestWorkflowExecutions :one
SELECT COUNT(*)
FROM v_test_workflow_execution_details
WHERE (organization_id = @organization_id AND environment_id = @environment_id)
    AND (COALESCE(@workflow_name::text, '') = '' OR workflow_name = @workflow_name::text)
    AND (COALESCE(@workflow_names::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR workflow_name = ANY(@workflow_names::text[]))
    AND (COALESCE(@text_search::text, '') = '' OR name ILIKE '%' || @text_search::text || '%')
    AND (COALESCE(@start_date::timestamptz, '1900-01-01'::timestamptz) = '1900-01-01'::timestamptz OR scheduled_at >= @start_date::timestamptz)
    AND (COALESCE(@end_date::timestamptz, '2100-01-01'::timestamptz) = '2100-01-01'::timestamptz OR scheduled_at <= @end_date::timestamptz)
    AND (COALESCE(@last_n_days::integer, 0) = 0 OR scheduled_at >= NOW() - (COALESCE(@last_n_days::integer, 0) || ' days')::interval)
    AND (COALESCE(@statuses::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR status = ANY(@statuses::text[]))
    AND (COALESCE(@runner_id::text, '') = '' OR runner_id = @runner_id::text)
    AND (COALESCE(@assigned, NULL) IS NULL OR 
         (@assigned::boolean = true AND runner_id IS NOT NULL AND runner_id != '') OR 
         (@assigned::boolean = false AND (runner_id IS NULL OR runner_id = '')))
    AND (COALESCE(@actor_name::text, '') = '' OR running_context->'actor'->>'name' = @actor_name::text)
    AND (COALESCE(@actor_type::text, '') = '' OR running_context->'actor'->>'type_' = @actor_type::text)
    AND (COALESCE(@group_id::text, '') = '' OR id = @group_id::text OR group_id = @group_id::text)
    AND (COALESCE(@initialized, NULL) IS NULL OR 
         (@initialized::boolean = true AND (status != 'queued' OR steps IS NOT NULL)) OR
         (@initialized::boolean = false AND status = 'queued' AND (steps IS NULL OR steps = '{}'::jsonb)))
    AND (COALESCE(@health_ranges::jsonb, '[]'::jsonb) = '[]'::jsonb OR 
         EXISTS (
             SELECT 1 FROM jsonb_array_elements(@health_ranges::jsonb) AS range_obj
             WHERE (workflow_status->>'health')::jsonb->>'overallHealth' IS NOT NULL 
               AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision >= (range_obj->>'min')::double precision
               AND ((workflow_status->>'health')::jsonb->>'overallHealth')::double precision <= (range_obj->>'max')::double precision
         )
    )
    AND (     
        (COALESCE(@tag_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR 
            (SELECT COUNT(*) FROM unnest(@tag_keys::text[]) AS key_condition
                WHERE 
                CASE 
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (tags ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        tags ? key_condition
                END
            ) = array_length(@tag_keys::text[], 1)
        )
        AND
        (COALESCE(@tag_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR 
            (SELECT COUNT(*) FROM unnest(@tag_conditions::text[]) AS cond
                WHERE tags->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@label_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_keys::text[]) AS key_condition
                WHERE
                CASE
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) > 0
        )
        AND
        (COALESCE(@label_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR
            (SELECT COUNT(*) FROM unnest(@label_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) > 0
        )
    )
    AND (
        (COALESCE(@selector_keys::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR 
            (SELECT COUNT(*) FROM unnest(@selector_keys::text[]) AS key_condition
                WHERE 
                CASE 
                    WHEN key_condition LIKE '%:not_exists' THEN
                        NOT (workflow_labels ? replace(key_condition, ':not_exists', ''))
                    ELSE
                        workflow_labels ? key_condition
                END
            ) = array_length(@selector_keys::text[], 1)
        )
        AND
        (COALESCE(@selector_conditions::text[], ARRAY[]::text[]) = ARRAY[]::text[] OR 
            (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond
                WHERE workflow_labels->>split_part(cond, '=', 1) = split_part(cond, '=', 2)
            ) = (SELECT COUNT(DISTINCT split_part(cond, '=', 1)) FROM unnest(@selector_conditions::text[]) AS cond)
        )
    );

-- name: GetTestWorkflowExecutionWithRunner :one
SELECT * FROM v_test_workflow_execution_details
WHERE (id = @id OR name = @id) AND runner_id = @runner_id AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: UpdateTestWorkflowExecutionResultStrict :one
UPDATE test_workflow_results 
SET 
    status = CASE 
        WHEN test_workflow_results.status = 'stopping' THEN 'stopping'
        ELSE @status
    END,
    predicted_status = CASE
        WHEN test_workflow_results.status = 'stopping' THEN test_workflow_results.predicted_status
        ELSE @predicted_status
    END,
    queued_at = @queued_at,
    started_at = @started_at,
    finished_at = @finished_at,
    duration = @duration,
    total_duration = @total_duration,
    duration_ms = @duration_ms,
    paused_ms = @paused_ms,
    total_duration_ms = @total_duration_ms,
    pauses = @pauses,
    initialization = @initialization,
    steps = @steps
FROM test_workflow_executions e
WHERE test_workflow_results.execution_id = @execution_id
    AND test_workflow_results.execution_id = e.id
    AND e.runner_id = @runner_id
    AND test_workflow_results.status IN (
        'assigned', 'starting', 'scheduling', 'running',
        'pausing', 'paused', 'resuming', 'stopping'
    )
RETURNING test_workflow_results.execution_id;

-- name: UpdateExecutionStatusAtStrict :exec
UPDATE test_workflow_executions 
SET status_at = CASE 
    WHEN @new_status != @old_status THEN @status_at 
    ELSE status_at 
END
WHERE id = @execution_id AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: FinishTestWorkflowExecutionResultStrict :one
UPDATE test_workflow_results 
SET 
    status = @status,
    predicted_status = @predicted_status,
    queued_at = @queued_at,
    started_at = @started_at,
    finished_at = @finished_at,
    duration = @duration,
    total_duration = @total_duration,
    duration_ms = @duration_ms,
    paused_ms = @paused_ms,
    total_duration_ms = @total_duration_ms,
    pauses = @pauses,
    initialization = @initialization,
    steps = @steps
FROM test_workflow_executions e
WHERE test_workflow_results.execution_id = @execution_id
    AND test_workflow_results.execution_id = e.id
    AND e.runner_id = @runner_id
    AND test_workflow_results.status IN (
        'queued', 'assigned', 'running', 'stopping',
        'starting', 'scheduling'
    )
RETURNING test_workflow_results.execution_id;

-- name: FinishExecutionStatusAtStrict :exec
UPDATE test_workflow_executions 
SET status_at = @finished_at
WHERE id = @execution_id AND (organization_id = @organization_id AND environment_id = @environment_id);
