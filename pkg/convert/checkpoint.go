package convert

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Task names double as the primary key of convert_checkpoints, so they are
// stable identifiers rather than display strings.
const (
	TaskExecutions = "executions"
	TaskSequences  = "sequences"
)

// AllTasks is the execution order. Executions run first because they are the
// expensive phase, and because sequences are cheap to redo if a run aborts.
var AllTasks = []string{TaskExecutions, TaskSequences}

// checkpoint is the persisted progress of one task.
type checkpoint struct {
	Task        string
	LastMongoID *bson.ObjectID
	Processed   int64
	Failed      int64
	Skipped     int64
	Completed   bool
}

// loadCheckpoint reads the stored progress for a task. A task that has never
// run yields a zero-valued checkpoint rather than an error.
func loadCheckpoint(ctx context.Context, db *pgxpool.Pool, task string) (*checkpoint, error) {
	const q = `SELECT last_mongo_id, processed_count, failed_count, skipped_count,
	                  completed_at IS NOT NULL
	             FROM convert_checkpoints
	            WHERE task = $1`

	cp := &checkpoint{Task: task}

	var rawID *string
	err := db.QueryRow(ctx, q, task).Scan(&rawID, &cp.Processed, &cp.Failed, &cp.Skipped, &cp.Completed)
	if errors.Is(err, pgx.ErrNoRows) {
		return cp, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint for task %q: %w", task, err)
	}

	if rawID != nil && *rawID != "" {
		id, err := bson.ObjectIDFromHex(*rawID)
		if err != nil {
			// A checkpoint we cannot parse is worse than none: resuming from it
			// would silently skip or re-copy an unknown span.
			return nil, fmt.Errorf("checkpoint for task %q holds an unparseable mongo id %q: %w", task, *rawID, err)
		}
		cp.LastMongoID = &id
	}

	return cp, nil
}

// saveCheckpoint upserts progress for a task. It takes a pgx.Tx rather than the
// pool on purpose: the caller commits it together with the batch's COPY
// statements so that progress can never run ahead of the data.
func saveCheckpoint(ctx context.Context, tx pgx.Tx, cp *checkpoint) error {
	const q = `INSERT INTO convert_checkpoints
	               (task, last_mongo_id, processed_count, failed_count, skipped_count, completed_at)
	           VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN NOW() ELSE NULL END)
	           ON CONFLICT (task) DO UPDATE SET
	               last_mongo_id   = EXCLUDED.last_mongo_id,
	               processed_count = EXCLUDED.processed_count,
	               failed_count    = EXCLUDED.failed_count,
	               skipped_count   = EXCLUDED.skipped_count,
	               completed_at    = EXCLUDED.completed_at,
	               updated_at      = NOW()`

	var rawID *string
	if cp.LastMongoID != nil {
		hex := cp.LastMongoID.Hex()
		rawID = &hex
	}

	if _, err := tx.Exec(ctx, q, cp.Task, rawID, cp.Processed, cp.Failed, cp.Skipped, cp.Completed); err != nil {
		return fmt.Errorf("failed to save checkpoint for task %q: %w", cp.Task, err)
	}
	return nil
}

// markCheckpointComplete stamps completed_at once a task has drained its source
// collection, so a later run reports "already done" instead of re-scanning.
func markCheckpointComplete(ctx context.Context, db *pgxpool.Pool, cp *checkpoint) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	cp.Completed = true
	if err := saveCheckpoint(ctx, tx, cp); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// clearCheckpoints drops all stored progress. Used by --reset, together with
// truncating the target tables.
func clearCheckpoints(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `DELETE FROM convert_checkpoints`); err != nil {
		return fmt.Errorf("failed to clear checkpoints: %w", err)
	}
	return nil
}
