-- name: UpsertAndIncrementExecutionSequence :one
INSERT INTO execution_sequences (name, number)
VALUES (@name, 1)
ON CONFLICT (name) DO UPDATE SET
    number = execution_sequences.number + 1,
    updated_at = NOW()
RETURNING name, number, created_at, updated_at;

-- name: DeleteExecutionSequence :exec
DELETE FROM execution_sequences WHERE name = @name;

-- name: DeleteExecutionSequences :exec
DELETE FROM execution_sequences WHERE name = ANY(@names::text[]);

-- name: DeleteAllExecutionSequences :exec
DELETE FROM execution_sequences;
