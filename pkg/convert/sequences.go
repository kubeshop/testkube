package convert

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/repository/sequence"
)

// sequenceDoc is the current shape of a sequences document, mirroring the
// unexported type in pkg/repository/sequence/mongo. There is no name field: the
// workflow name lives in _id as "<executionType>-<name>".
type sequenceDoc struct {
	ID            string                 `bson:"_id"`
	Number        int32                  `bson:"number"`
	ExecutionType sequence.ExecutionType `bson:"executionType"`

	// LegacyName is set on pre-executionType documents, which were keyed by a
	// name field carrying a "ts-" prefix for test suites and no prefix otherwise.
	LegacyName string `bson:"name"`
}

// sequenceRow is one resolved counter ready for the target table.
type sequenceRow struct {
	Name   string
	Number int32
}

// sequenceMigrator copies execution counters into execution_sequences.
//
// Only test workflow counters are migrated. The Postgres sequence repository
// ignores sequence.ExecutionType entirely (pkg/repository/sequence/postgres
// takes it as a blank parameter) and keys solely on name, so carrying a legacy
// "t-foo" across would collide with "tw-foo" in that type-less keyspace. The
// legacy Test and TestSuite entities have no Postgres repositories at all, so
// their counters have no consumer either way.
type sequenceMigrator struct {
	mongo  *mongo.Collection
	pg     *pgxpool.Pool
	log    *zap.SugaredLogger
	config Config
}

func newSequenceMigrator(db *mongo.Database, pg *pgxpool.Pool, log *zap.SugaredLogger, cfg Config) *sequenceMigrator {
	return &sequenceMigrator{
		mongo:  db.Collection(sequencesCollection),
		pg:     pg,
		log:    log,
		config: cfg,
	}
}

// Migrate copies every test workflow counter across. The collection holds one
// document per workflow, so it is read and written in a single batch.
func (m *sequenceMigrator) Migrate(ctx context.Context) (*Stats, error) {
	stats := newStats(TaskSequences)
	defer stats.finish()

	total, err := m.mongo.CountDocuments(ctx, bson.M{})
	if err != nil {
		return stats, fmt.Errorf("failed to count sequences: %w", err)
	}
	stats.Total = total

	cp, err := loadCheckpoint(ctx, m.pg, TaskSequences)
	if err != nil {
		return stats, err
	}
	if cp.Completed {
		m.log.Info("Sequences already migrated; re-running to fold in any newer counters")
	}

	rows, err := m.read(ctx, stats)
	if err != nil {
		return stats, err
	}

	if err := m.write(ctx, rows, stats, cp); err != nil {
		return stats, err
	}

	// This task rereads the whole collection on every run rather than resuming
	// from a position, so its own counters already cover every document. Adding
	// the checkpoint's totals here would count the same skipped documents once per
	// run and drive the expected row count negative.
	stats.CumulativeFailed = stats.Failed
	stats.CumulativeSkipped = stats.Skipped

	stats.Print(m.log, nil)
	return stats, nil
}

// read resolves the collection into target rows, reporting every document it
// declines to migrate.
func (m *sequenceMigrator) read(ctx context.Context, stats *Stats) ([]sequenceRow, error) {
	cursor, err := m.mongo.Find(ctx, bson.M{},
		options.Find().SetBatchSize(int32(m.config.ReadBatchSize)))
	if err != nil {
		return nil, fmt.Errorf("failed to open sequences cursor: %w", err)
	}
	defer cursor.Close(ctx) //nolint:errcheck // best effort on the read path

	// Fold duplicates -- a legacy name-keyed document and a current _id-keyed one
	// can describe the same workflow -- keeping the highest counter so numbering
	// can never move backwards.
	highest := make(map[string]int32)

	for cursor.Next(ctx) {
		var doc sequenceDoc
		if err := cursor.Decode(&doc); err != nil {
			stats.addError("failed to decode sequence: %v", err)
			if !m.config.SkipErrors {
				return nil, fmt.Errorf("failed to decode sequence: %w", err)
			}
			continue
		}

		name, ok := m.resolveName(&doc, stats)
		if !ok {
			continue
		}

		if current, seen := highest[name]; !seen || doc.Number > current {
			highest[name] = doc.Number
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("sequences cursor failed: %w", err)
	}

	rows := make([]sequenceRow, 0, len(highest))
	for name, number := range highest {
		rows = append(rows, sequenceRow{Name: name, Number: number})
	}
	return rows, nil
}

// resolveName extracts the workflow name from a sequences document, or reports
// why the document is not a test workflow counter.
func (m *sequenceMigrator) resolveName(doc *sequenceDoc, stats *Stats) (string, bool) {
	prefix := string(sequence.ExecutionTypeTestWorkflow) + "-"

	// Legacy documents carry a name and no executionType. A "ts-" prefix marks a
	// test suite counter; anything else was a plain test workflow counter.
	if doc.ExecutionType == "" {
		if doc.LegacyName == "" {
			stats.addSkip(m.log, "Skipping sequence %q: no executionType and no name", doc.ID)
			return "", false
		}
		if strings.HasPrefix(doc.LegacyName, string(sequence.ExecutionTypeTestSuite)+"-") {
			stats.addSkip(m.log, "Skipping legacy test suite sequence %q", doc.LegacyName)
			return "", false
		}
		return doc.LegacyName, true
	}

	if doc.ExecutionType != sequence.ExecutionTypeTestWorkflow {
		stats.addSkip(m.log, "Skipping sequence %q (executionType=%s): no Postgres consumer",
			doc.ID, doc.ExecutionType)
		return "", false
	}

	if !strings.HasPrefix(doc.ID, prefix) {
		stats.addSkip(m.log, "Skipping sequence with unexpected _id format %q (executionType=%s)",
			doc.ID, doc.ExecutionType)
		return "", false
	}

	name := strings.TrimPrefix(doc.ID, prefix)
	if name == "" {
		stats.addSkip(m.log, "Skipping sequence %q: empty workflow name after removing the %q prefix",
			doc.ID, prefix)
		return "", false
	}

	return name, true
}

// write upserts the counters.
//
// COPY cannot express ON CONFLICT, so rows land in a temporary table first and
// are then merged with GREATEST. That keeps the phase re-runnable and, more
// importantly, guarantees a counter never moves backwards -- if it did,
// executions scheduled after the migration would reuse the numbers of
// migrated ones.
func (m *sequenceMigrator) write(ctx context.Context, rows []sequenceRow, stats *Stats, cp *checkpoint) error {
	if len(rows) == 0 {
		m.log.Info("No test workflow sequences to migrate")
		return nil
	}

	var buf bytes.Buffer
	for i := range rows {
		if _, err := fmt.Fprintf(&buf, "%s\t%d\n", escapeCopyValue(rows[i].Name), rows[i].Number); err != nil {
			return fmt.Errorf("failed to serialize sequence %q: %w", rows[i].Name, err)
		}
	}

	if m.config.DryRun {
		stats.Processed += int64(len(rows))
		stats.Batches++
		m.log.Debugf("dry run: would upsert %d sequences", len(rows))
		return nil
	}

	tx, err := m.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin sequence transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx,
		`CREATE TEMP TABLE convert_sequence_import (name TEXT, number INTEGER) ON COMMIT DROP`); err != nil {
		return fmt.Errorf("failed to create sequence staging table: %w", err)
	}

	if _, err := copyFrom(ctx, tx, "convert_sequence_import", bytes.NewReader(buf.Bytes()), "name", "number"); err != nil {
		return err
	}

	// organization_id and environment_id are the empty string because OSS never
	// populates them; they are part of the primary key, so they must be written
	// explicitly here rather than left to the column default.
	tag, err := tx.Exec(ctx, `
		INSERT INTO execution_sequences (name, organization_id, environment_id, number)
		SELECT name, '', '', number FROM convert_sequence_import
		ON CONFLICT (name, organization_id, environment_id) DO UPDATE
		    SET number     = GREATEST(execution_sequences.number, EXCLUDED.number),
		        updated_at = NOW()`)
	if err != nil {
		return fmt.Errorf("failed to upsert sequences: %w", err)
	}

	cp.Processed = int64(len(rows))
	cp.Failed = stats.Failed
	cp.Skipped = stats.Skipped
	cp.Completed = true
	if err := saveCheckpoint(ctx, tx, cp); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit sequences: %w", err)
	}

	stats.Processed += tag.RowsAffected()
	stats.Batches++
	return nil
}

// truncateSequences clears the counter table.
func truncateSequences(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `TRUNCATE execution_sequences`); err != nil {
		return fmt.Errorf("failed to truncate sequences: %w", err)
	}
	return nil
}

// countSequenceRows is used by the verification pass.
func countSequenceRows(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	var n int64
	if err := db.QueryRow(ctx, `SELECT count(*) FROM execution_sequences`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
