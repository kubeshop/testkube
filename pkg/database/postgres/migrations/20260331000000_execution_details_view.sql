-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW v_test_workflow_execution_details AS
SELECT
    e.id, e.group_id, e.runner_id, e.runner_target, e.runner_original_target,
    e.name, e.namespace, e.number, e.scheduled_at, e.assigned_at, e.status_at,
    e.test_workflow_execution_name, e.disable_webhooks, e.tags, e.running_context,
    e.config_params, e.runtime, e.silent_mode, e.created_at, e.updated_at,
    e.organization_id, e.environment_id,
    r.status, r.predicted_status, r.queued_at, r.started_at, r.finished_at,
    r.duration, r.total_duration, r.duration_ms, r.paused_ms, r.total_duration_ms,
    r.pauses, r.initialization, r.steps,
    w.name as workflow_name, w.namespace as workflow_namespace, w.description as workflow_description,
    w.labels as workflow_labels, w.annotations as workflow_annotations, w.created as workflow_created,
    w.updated as workflow_updated, w.spec as workflow_spec, w.read_only as workflow_read_only,
    w.status as workflow_status,
    rw.name as resolved_workflow_name, rw.namespace as resolved_workflow_namespace,
    rw.description as resolved_workflow_description, rw.labels as resolved_workflow_labels,
    rw.annotations as resolved_workflow_annotations, rw.created as resolved_workflow_created,
    rw.updated as resolved_workflow_updated, rw.spec as resolved_workflow_spec,
    rw.read_only as resolved_workflow_read_only, rw.status as resolved_workflow_status,
    COALESCE(
        (SELECT json_agg(
            json_build_object(
                'id', s.id, 'ref', s.ref, 'name', s.name, 'category', s.category,
                'optional', s.optional, 'negative', s.negative, 'parent_id', s.parent_id
            ) ORDER BY s.sig_order
        ) FROM test_workflow_signatures s WHERE s.execution_id = e.id),
        '[]'::json
    )::json as signatures_json,
    COALESCE(
        (SELECT json_agg(
            json_build_object(
                'id', o.id, 'ref', o.ref, 'name', o.name, 'value', o.value
            ) ORDER BY o.out_order
        ) FROM test_workflow_outputs o WHERE o.execution_id = e.id),
        '[]'::json
    )::json as outputs_json,
    COALESCE(
        (SELECT json_agg(
            json_build_object(
                'id', rep.id, 'ref', rep.ref, 'kind', rep.kind,
                'file', rep.file, 'summary', rep.summary
            ) ORDER BY rep.rep_order
        ) FROM test_workflow_reports rep WHERE rep.execution_id = e.id),
        '[]'::json
    )::json as reports_json,
    ra.global as resource_aggregations_global,
    ra.step as resource_aggregations_step
FROM test_workflow_executions e
LEFT JOIN test_workflow_results r ON e.id = r.execution_id
LEFT JOIN test_workflows w ON e.id = w.execution_id AND w.workflow_type = 'workflow'
LEFT JOIN test_workflows rw ON e.id = rw.execution_id AND rw.workflow_type = 'resolved_workflow'
LEFT JOIN test_workflow_resource_aggregations ra ON e.id = ra.execution_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS v_test_workflow_execution_details;
-- +goose StatementEnd
