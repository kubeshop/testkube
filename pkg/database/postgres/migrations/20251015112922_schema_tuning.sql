-- +goose Up
-- +goose StatementBegin
-- Drop created_at and updated_at columns
ALTER TABLE test_workflows DROP COLUMN created_at;
ALTER TABLE test_workflows DROP COLUMN updated_at;
ALTER TABLE test_workflows DROP COLUMN id;
ALTER TABLE test_workflows ADD COLUMN id UUID DEFAULT gen_random_uuid() PRIMARY KEY;

ALTER TABLE test_workflow_outputs ADD COLUMN out_order INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_test_workflow_outputs_out_order ON test_workflow_outputs(out_order);
UPDATE test_workflow_outputs SET out_order = id;
ALTER TABLE test_workflow_outputs DROP COLUMN id;
ALTER TABLE test_workflow_outputs ADD COLUMN id UUID DEFAULT gen_random_uuid() PRIMARY KEY;

ALTER TABLE test_workflow_reports ADD COLUMN rep_order INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_test_workflow_reports_rep_order ON test_workflow_reports(rep_order);
UPDATE test_workflow_reports SET rep_order = id;
ALTER TABLE test_workflow_reports DROP COLUMN id;
ALTER TABLE test_workflow_reports ADD COLUMN id UUID DEFAULT gen_random_uuid() PRIMARY KEY;

ALTER TABLE test_workflow_signatures ADD COLUMN sig_order INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_test_workflow_signatures_sig_order ON test_workflow_signatures(sig_order);
UPDATE test_workflow_signatures SET sig_order = id;
ALTER TABLE test_workflow_signatures ADD COLUMN uuid_id UUID DEFAULT gen_random_uuid();
ALTER TABLE test_workflow_signatures ADD COLUMN parent_uuid UUID;
UPDATE test_workflow_signatures SET parent_uuid = t.uuid_id FROM test_workflow_signatures t WHERE test_workflow_signatures.parent_id = t.id;
ALTER TABLE test_workflow_signatures DROP COLUMN parent_id;
ALTER TABLE test_workflow_signatures DROP COLUMN id;
ALTER TABLE test_workflow_signatures RENAME COLUMN parent_uuid TO parent_id;
ALTER TABLE test_workflow_signatures RENAME COLUMN uuid_id TO id;
ALTER TABLE test_workflow_signatures ADD PRIMARY KEY (id);
ALTER TABLE test_workflow_signatures ADD CONSTRAINT test_workflow_signatures_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES test_workflow_signatures(id) ON DELETE CASCADE;
CREATE INDEX idx_test_workflow_signatures_parent_id ON test_workflow_signatures(parent_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE test_workflows ADD COLUMN created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
ALTER TABLE test_workflows ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
ALTER TABLE test_workflows DROP COLUMN id;
ALTER TABLE test_workflows ADD COLUMN id SERIAL;
ALTER TABLE test_workflows ADD PRIMARY KEY (id);

DROP INDEX idx_test_workflow_outputs_out_order;
ALTER TABLE test_workflow_outputs DROP COLUMN out_order;
ALTER TABLE test_workflow_outputs DROP COLUMN id;
ALTER TABLE test_workflow_outputs ADD COLUMN id SERIAL;
ALTER TABLE test_workflow_outputs ADD PRIMARY KEY (id);

DROP INDEX idx_test_workflow_reports_rep_order;
ALTER TABLE test_workflow_reports DROP COLUMN rep_order;
ALTER TABLE test_workflow_reports DROP COLUMN id;
ALTER TABLE test_workflow_reports ADD COLUMN id SERIAL;
ALTER TABLE test_workflow_reports ADD PRIMARY KEY (id);

DROP INDEX idx_test_workflow_signatures_sig_order;
ALTER TABLE test_workflow_signatures DROP COLUMN sig_order;
ALTER TABLE test_workflow_signatures ADD COLUMN serial_id SERIAL;
ALTER TABLE test_workflow_signatures ADD COLUMN parent_serial INTEGER;
UPDATE test_workflow_signatures SET parent_serial = t.serial_id FROM test_workflow_signatures t WHERE test_workflow_signatures.parent_id = t.id;
ALTER TABLE test_workflow_signatures DROP COLUMN parent_id;
ALTER TABLE test_workflow_signatures DROP COLUMN id;
ALTER TABLE test_workflow_signatures RENAME COLUMN parent_serial TO parent_id;
ALTER TABLE test_workflow_signatures RENAME COLUMN serial_id TO id;
ALTER TABLE test_workflow_signatures ADD PRIMARY KEY (id);
ALTER TABLE test_workflow_signatures ADD CONSTRAINT test_workflow_signatures_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES test_workflow_signatures(id) ON DELETE CASCADE;
CREATE INDEX idx_test_workflow_signatures_parent_id ON test_workflow_signatures(parent_id);
-- +goose StatementEnd
