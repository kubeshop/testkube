-- name: UpsertAndIncrementExecutionSequence :one
INSERT INTO execution_sequences (name, number, organization_id, environment_id)
VALUES (@name, 1, @organization_id, @environment_id)
ON CONFLICT (name, organization_id, environment_id) DO UPDATE SET
    number = execution_sequences.number + 1,
    updated_at = NOW()
RETURNING name, number, created_at, updated_at, organization_id, environment_id;

-- name: DeleteExecutionSequence :exec
DELETE FROM execution_sequences WHERE name = @name AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: DeleteExecutionSequences :exec
DELETE FROM execution_sequences WHERE name = ANY(@names::text[]) AND (organization_id = @organization_id AND environment_id = @environment_id);

-- name: DeleteAllExecutionSequences :exec
DELETE FROM execution_sequences WHERE organization_id = @organization_id AND environment_id = @environment_id;
