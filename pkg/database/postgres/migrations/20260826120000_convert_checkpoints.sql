-- +goose Up
-- +goose StatementBegin
-- Progress state for the Mongo -> Postgres convert tool (cmd/convert).
--
-- One row per migration task. last_mongo_id is the hex ObjectID of the last
-- source document included in a committed batch; the tool resumes by asking
-- Mongo for documents with a strictly greater _id. The checkpoint is written in
-- the same transaction as the COPY statements for its batch, which is what
-- makes the migration crash-safe and exactly-once.
--
-- This table is written only by the convert tool; the API server never reads it.
CREATE TABLE IF NOT EXISTS convert_checkpoints (
    task            VARCHAR(64) PRIMARY KEY,
    last_mongo_id   VARCHAR(64),
    processed_count BIGINT      NOT NULL DEFAULT 0,
    failed_count    BIGINT      NOT NULL DEFAULT 0,
    skipped_count   BIGINT      NOT NULL DEFAULT 0,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS convert_checkpoints;
-- +goose StatementEnd
