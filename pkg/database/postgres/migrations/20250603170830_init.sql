
-- +goose Up
-- +goose StatementBegin
-- Create the main table for TestWorkflowExecution
-- Main table for TestWorkflowExecution (simplified)
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
    test_workflow_execution_name VARCHAR(255),
    disable_webhooks BOOLEAN DEFAULT FALSE,
    tags JSONB,
    running_context JSONB,
    config_params JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- TestWorkflowSignature table
CREATE TABLE test_workflow_signatures (
    id SERIAL PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL REFERENCES test_workflow_executions(id) ON DELETE CASCADE,
    ref VARCHAR(255),
    name VARCHAR(255),
    category VARCHAR(255),
    optional BOOLEAN DEFAULT FALSE,
    negative BOOLEAN DEFAULT FALSE,
    parent_id INTEGER REFERENCES test_workflow_signatures(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- TestWorkflowResult table
CREATE TABLE test_workflow_results (
    execution_id VARCHAR(255) PRIMARY KEY REFERENCES test_workflow_executions(id) ON DELETE CASCADE,
    status VARCHAR(50),
    predicted_status VARCHAR(50),
    queued_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    duration VARCHAR(100),
    total_duration VARCHAR(100),
    duration_ms INTEGER DEFAULT 0,
    paused_ms INTEGER DEFAULT 0,
    total_duration_ms INTEGER DEFAULT 0,
    pauses JSONB,
    initialization JSONB,
    steps JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- TestWorkflowOutput table
CREATE TABLE test_workflow_outputs (
    id SERIAL PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL REFERENCES test_workflow_executions(id) ON DELETE CASCADE,
    ref VARCHAR(255),
    name VARCHAR(255),
    value JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- TestWorkflowReport table
CREATE TABLE test_workflow_reports (
    id SERIAL PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL REFERENCES test_workflow_executions(id) ON DELETE CASCADE,
    ref VARCHAR(255),
    kind VARCHAR(255),
    file VARCHAR(500),
    summary JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- TestWorkflowExecutionResourceAggregations table
CREATE TABLE test_workflow_resource_aggregations (
    execution_id VARCHAR(255) PRIMARY KEY REFERENCES test_workflow_executions(id) ON DELETE CASCADE,
    global JSONB,
    step JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- TestWorkflow table (for workflow field)
CREATE TABLE test_workflows (
    id SERIAL PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL REFERENCES test_workflow_executions(id) ON DELETE CASCADE,
    workflow_type VARCHAR(20) NOT NULL, -- 'workflow' or 'resolved_workflow'
    name VARCHAR(255),
    namespace VARCHAR(255),
    description TEXT,
    labels JSONB,
    annotations JSONB,
    created TIMESTAMP WITH TIME ZONE,
    updated TIMESTAMP WITH TIME ZONE,
    spec JSONB,
    read_only BOOLEAN DEFAULT FALSE,
    status JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(execution_id, workflow_type)
);

-- Create the main table for Config
CREATE TABLE configs (
    id VARCHAR(255) PRIMARY KEY,
    cluster_id VARCHAR(255) NOT NULL,
    enable_telemetry BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_test_workflow_executions_group_id ON test_workflow_executions(group_id);
CREATE INDEX idx_test_workflow_executions_runner_id ON test_workflow_executions(runner_id);
CREATE INDEX idx_test_workflow_executions_name ON test_workflow_executions(name);
CREATE INDEX idx_test_workflow_executions_namespace ON test_workflow_executions(namespace);
CREATE INDEX idx_test_workflow_executions_scheduled_at ON test_workflow_executions(scheduled_at DESC);
CREATE INDEX idx_test_workflow_executions_status_at ON test_workflow_executions(status_at DESC);
CREATE INDEX idx_test_workflow_executions_tags ON test_workflow_executions USING GIN (tags);

CREATE INDEX idx_test_workflow_signatures_execution_id ON test_workflow_signatures(execution_id);
CREATE INDEX idx_test_workflow_signatures_parent_id ON test_workflow_signatures(parent_id);

CREATE INDEX idx_test_workflow_results_status ON test_workflow_results(status);
CREATE INDEX idx_test_workflow_results_finished_at ON test_workflow_results(finished_at DESC);

CREATE INDEX idx_test_workflow_outputs_execution_id ON test_workflow_outputs(execution_id);

CREATE INDEX idx_test_workflow_reports_execution_id ON test_workflow_reports(execution_id);

CREATE INDEX idx_test_workflow_resource_aggregations_execution_id ON test_workflow_resource_aggregations(execution_id);

CREATE INDEX idx_test_workflows_execution_id ON test_workflows(execution_id);
CREATE INDEX idx_test_workflows_workflow_type ON test_workflows(workflow_type);
CREATE INDEX idx_test_workflows_name ON test_workflows(name);

CREATE INDEX idx_configs_cluster_id ON configs(cluster_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_configs_cluster_id;

DROP INDEX IF EXISTS idx_test_workflows_execution_id;
DROP INDEX IF EXISTS idx_test_workflows_workflow_type;
DROP INDEX IF EXISTS idx_test_workflows_name;

DROP INDEX IF EXISTS idx_test_workflow_resource_aggregations_execution_id;

DROP INDEX IF EXISTS idx_test_workflow_reports_execution_id;

DROP INDEX IF EXISTS idx_test_workflow_outputs_execution_id;

DROP INDEX IF EXISTS idx_test_workflow_results_status;
DROP INDEX IF EXISTS idx_test_workflow_results_finished_at;

DROP INDEX IF EXISTS idx_test_workflow_signatures_execution_id;
DROP INDEX IF EXISTS idx_test_workflow_signatures_parent_id;

DROP INDEX IF EXISTS idx_test_workflow_executions_group_id;
DROP INDEX IF EXISTS idx_test_workflow_executions_runner_id;
DROP INDEX IF EXISTS idx_test_workflow_executions_name;
DROP INDEX IF EXISTS idx_test_workflow_executions_namespace;
DROP INDEX IF EXISTS idx_test_workflow_executions_scheduled_at;
DROP INDEX IF EXISTS idx_test_workflow_executions_status_at;
DROP INDEX IF EXISTS idx_test_workflow_executions_tags;

DROP TABLE configs;
DROP TABLE IF EXISTS test_workflows;
DROP TABLE IF EXISTS test_workflow_resource_aggregations;
DROP TABLE IF EXISTS test_workflow_reports;
DROP TABLE IF EXISTS test_workflow_outputs;
DROP TABLE IF EXISTS test_workflow_results;
DROP TABLE IF EXISTS test_workflow_signatures;
DROP TABLE IF EXISTS test_workflow_executions;
-- +goose StatementEnd
