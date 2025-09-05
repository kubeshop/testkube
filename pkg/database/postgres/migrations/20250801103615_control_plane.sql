-- +goose Up
-- +goose StatementBegin
ALTER TABLE test_workflow_executions ADD COLUMN organization_id VARCHAR(255) DEFAULT '' NOT NULL;
ALTER TABLE test_workflow_executions ADD COLUMN environment_id VARCHAR(255) DEFAULT '' NOT NULL;

ALTER TABLE execution_sequences ADD COLUMN organization_id VARCHAR(255) DEFAULT '' NOT NULL;
ALTER TABLE execution_sequences ADD COLUMN environment_id VARCHAR(255) DEFAULT '' NOT NULL;
ALTER TABLE execution_sequences DROP CONSTRAINT execution_sequences_pkey;
ALTER TABLE execution_sequences ADD PRIMARY KEY (name, organization_id, environment_id);

-- Create indexes
CREATE INDEX idx_test_workflow_executions_organization_id ON test_workflow_executions(organization_id);
CREATE INDEX idx_test_workflow_executions_environment_id ON test_workflow_executions(environment_id);

CREATE INDEX idx_execution_sequences_organization_id ON execution_sequences(organization_id);
CREATE INDEX idx_execution_sequences_environment_id ON execution_sequences(environment_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_execution_sequences_organization_id;
DROP INDEX IF EXISTS idx_execution_sequences_environment_id;

DROP INDEX IF EXISTS idx_test_workflow_executions_organization_id;
DROP INDEX IF EXISTS idx_test_workflow_executions_environment_id;

ALTER TABLE execution_sequences DROP COLUMN organization_id;
ALTER TABLE execution_sequences DROP COLUMN environment_id;
ALTER TABLE execution_sequences DROP CONSTRAINT execution_sequences_pkey;
ALTER TABLE execution_sequences ADD PRIMARY KEY (name);

ALTER TABLE test_workflow_executions DROP COLUMN organization_id;
ALTER TABLE test_workflow_executions DROP COLUMN environment_id;
-- +goose StatementEnd
