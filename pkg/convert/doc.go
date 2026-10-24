// Package convert migrates Testkube control-plane data from MongoDB to
// PostgreSQL. cmd/convert is its CLI and flags.
//
// It exists because an OSS installation that switches API_MONGO_DSN for
// API_POSTGRES_DSN otherwise starts from an empty database: historical Test
// Workflow executions vanish, and execution numbering restarts at 1 so new
// executions collide with the names of old ones.
//
// Two collections carry data worth moving, as two tasks:
//
//   - executions copies testworkflowresults into the seven test_workflow_*
//     tables (executions.go, executions_row.go).
//   - sequences copies Test Workflow counters into execution_sequences
//     (sequences.go), and never moves a counter backwards.
//
// Everything else is deliberately out of scope. Execution logs, outputs and
// artifacts live in object storage, not Mongo -- only their references travel
// with the execution. Test workflow and template definitions are Kubernetes
// custom resources. Cluster configuration lives in a ConfigMap. The triggers
// collection holds a leader-election lease with a one-minute lifetime that the
// API server reacquires on startup, so copying it across would move a stale row
// and nothing else.
//
// # Rows are written by hand
//
// Rows are written as COPY text, not through the repository. That makes every
// schema migration touching test_workflow_executions or its child tables a
// change here too: add the column to the matching *Columns list and write*Row
// serializer in executions_row.go, projected the way
// pkg/repository/testworkflow/postgres writes it. A column the converter leaves
// out is silently NULL for every migrated row.
//
// TestConvertExecutions_Integration reads each execution back through both
// repositories and compares them, which catches the omission only if the
// fixture in buildExecution sets the field. Both repositories synthesize some
// fields on read (lineage through EffectiveLineage), so a fixture that leaves
// such a field unset round-trips even when the column was dropped.
//
// # Running it
//
// Each batch commits in the same transaction as its row in convert_checkpoints
// (migration 20260826120000_convert_checkpoints.sql, written only by this
// tool), so an interrupted run resumes after the last committed batch and every
// document is written exactly once. --reset --yes truncates the target and
// clears the checkpoints. --dry-run serializes everything and writes nothing,
// but still applies schema migrations, because the checkpoint lookup and the
// verification read those tables.
//
// The tool migrates the schema itself, with goose's WithAllowOutofOrder(true)
// like the API server, and an out-of-date schema is fatal here rather than a
// warning: the COPY statements name columns that may not exist yet.
//
// After the summary, the tool exits non-zero if any document failed or the run
// raised a warning, such as a verification count mismatch (ErrIncomplete), so a
// Job is marked failed.
//
// The tool does not use internal/config. Each flag in cmd/convert/main.go falls
// back to an env var: it reuses the API server's API_MONGO_*, API_POSTGRES_DSN,
// SKIP_DB_CREATION and DISABLE_POSTGRES_MIGRATIONS, and adds its own
// CONVERT_BATCH_SIZE, CONVERT_READ_BATCH_SIZE, CONVERT_DRY_RUN, CONVERT_RESET,
// CONVERT_RESET_CONFIRMED, CONVERT_SKIP_ERRORS, CONVERT_SKIP and
// CONVERT_VERIFY. MongoDB TLS material is passed as file paths, because the
// tool has no cluster client to read a Secret with.
//
// The testkube chart runs it as the convert Job (templates/convert-job.yaml,
// convert.* values), disabled by default. It is deliberately not a Helm hook:
// the operator triggers the cutover. The Job reads both DSNs from
// testkube-api.mongodb and testkube-api.postgresql, and a retry resumes from
// the checkpoint.
package convert
