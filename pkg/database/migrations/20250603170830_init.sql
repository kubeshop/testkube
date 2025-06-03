
-- +goose Up
-- +goose StatementBegin
-- Create the main table for TestWorkflowExecution
CREATE TABLE test_workflow_executions (
    id VARCHAR(255) PRIMARY KEY,
    group_id VARCHAR(255),
    runner_id VARCHAR(255),
    runner_target JSONB,
    runner_original_target JSONB,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255),
    number INTEGER,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    assigned_at TIMESTAMP WITH TIME ZONE,
    status_at TIMESTAMP WITH TIME ZONE,
    signature JSONB,
    result JSONB,
    output JSONB,
    reports JSONB,
    resource_aggregations JSONB,
    workflow JSONB,
    resolved_workflow JSONB,
    test_workflow_execution_name VARCHAR(255),
    disable_webhooks BOOLEAN DEFAULT FALSE,
    tags JSONB,
    running_context JSONB,
    config_params JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX idx_test_workflow_executions_group_id ON test_workflow_executions(group_id);
CREATE INDEX idx_test_workflow_executions_runner_id ON test_workflow_executions(runner_id);
CREATE INDEX idx_test_workflow_executions_name ON test_workflow_executions(name);
CREATE INDEX idx_test_workflow_executions_namespace ON test_workflow_executions(namespace);
CREATE INDEX idx_test_workflow_executions_number ON test_workflow_executions(number);
CREATE INDEX idx_test_workflow_executions_scheduled_at ON test_workflow_executions(scheduled_at DESC);
CREATE INDEX idx_test_workflow_executions_status_at ON test_workflow_executions(status_at DESC);
CREATE INDEX idx_test_workflow_executions_workflow_name ON test_workflow_executions USING GIN ((workflow->>'name'));
CREATE INDEX idx_test_workflow_executions_result_status ON test_workflow_executions USING GIN ((result->>'status'));
CREATE INDEX idx_test_workflow_executions_tags ON test_workflow_executions USING GIN (tags);
CREATE INDEX idx_test_workflow_executions_running_context_actor ON test_workflow_executions USING GIN ((running_context->'actor'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_test_workflow_executions_group_id;
DROP INDEX idx_test_workflow_executions_runner_id;
DROP INDEX idx_test_workflow_executions_name;
DROP INDEX idx_test_workflow_executions_namespace;
DROP INDEX idx_test_workflow_executions_number;
DROP INDEX idx_test_workflow_executions_scheduled_at;
DROP INDEX idx_test_workflow_executions_status_at;
DROP INDEX idx_test_workflow_executions_workflow_name;
DROP INDEX idx_test_workflow_executions_result_status;
DROP INDEX idx_test_workflow_executions_tags;
DROP INDEX idx_test_workflow_executions_running_context_actor;
DROP TABLE IF EXISTS test_workflow_executions;
-- +goose StatementEnd
