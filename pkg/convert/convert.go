// Package convert migrates Testkube control-plane data from MongoDB to
// PostgreSQL.
//
// It exists because an OSS installation that switches API_MONGO_DSN for
// API_POSTGRES_DSN otherwise starts from an empty database: historical Test
// Workflow executions vanish, and execution numbering restarts at 1 so new
// executions collide with the names of old ones.
//
// Two collections carry data worth moving:
//
//   - testworkflowresults, into the seven test_workflow_* tables
//   - sequences, into execution_sequences
//
// Everything else is deliberately out of scope. Execution logs, outputs and
// artifacts live in object storage, not Mongo -- only their references travel
// with the execution. Test workflow and template definitions are Kubernetes
// custom resources. Cluster configuration lives in a ConfigMap. The triggers
// collection holds a leader-election lease with a one-minute lifetime that the
// API server reacquires on startup, so copying it across would move a stale row
// and nothing else.
package convert

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"

	sequencemongo "github.com/kubeshop/testkube/pkg/repository/sequence/mongo"
	testworkflowmongo "github.com/kubeshop/testkube/pkg/repository/testworkflow/mongo"
)

// Source collection names, referenced from the repositories that own them so
// the migrator cannot drift away from where the data actually lives.
const (
	executionsCollection = testworkflowmongo.CollectionName
	sequencesCollection  = sequencemongo.CollectionSequences
)

// Default tuning. The batch size is deliberately far below the cloud tool's
// 100k: a batch is held in memory and committed as one transaction, so a
// smaller value keeps both the footprint and the lock duration predictable
// while still giving COPY plenty to work with.
const (
	DefaultBatchSize     = 1000
	DefaultReadBatchSize = 500
)

// Config controls one conversion run.
type Config struct {
	// BatchSize is how many executions are committed per transaction.
	BatchSize int
	// ReadBatchSize is the MongoDB cursor batch size.
	ReadBatchSize int

	// DryRun reads and serializes everything but commits nothing. Unlike the
	// cloud tool, which skips serialization entirely, this exercises the full
	// mapping so a dry run actually validates the data.
	DryRun bool

	// Reset truncates the target tables and clears all checkpoints first.
	Reset bool

	// SkipErrors keeps going past documents that cannot be migrated instead of
	// aborting the run.
	SkipErrors bool

	// Skip names tasks to leave out. See AllTasks.
	Skip []string

	// Verify compares source and target counts after migrating.
	Verify bool
}

// WithDefaults fills in unset tuning values.
func (c Config) WithDefaults() Config {
	if c.BatchSize <= 0 {
		c.BatchSize = DefaultBatchSize
	}
	if c.ReadBatchSize <= 0 {
		c.ReadBatchSize = DefaultReadBatchSize
	}
	return c
}

// skips reports whether a task was excluded.
func (c Config) skips(task string) bool {
	for _, s := range c.Skip {
		if strings.EqualFold(strings.TrimSpace(s), task) {
			return true
		}
	}
	return false
}

// Result is the outcome of a run.
type Result struct {
	Stats    map[string]*Stats
	Warnings []string
	Duration time.Duration
}

// Failed reports whether anything went wrong: a record that could not be
// migrated, or a verification mismatch. Callers use it to pick an exit code.
func (r *Result) Failed() bool {
	if len(r.Warnings) > 0 {
		return true
	}
	for _, s := range r.Stats {
		if s.Failed > 0 {
			return true
		}
	}
	return false
}

// Converter runs the migration.
type Converter struct {
	mongo  *mongo.Database
	pg     *pgxpool.Pool
	log    *zap.SugaredLogger
	config Config
}

// New builds a Converter over an already-connected pair of databases. The caller
// owns both connections, and is expected to have applied the Postgres migrations
// already -- the target tables, including convert_checkpoints, must exist.
func New(mongoDB *mongo.Database, pg *pgxpool.Pool, log *zap.SugaredLogger, cfg Config) *Converter {
	return &Converter{mongo: mongoDB, pg: pg, log: log, config: cfg.WithDefaults()}
}

// Run performs the migration and returns per-task statistics. An error means
// the run aborted; a Result whose Failed reports true means it completed but
// something needs attention.
func (c *Converter) Run(ctx context.Context) (*Result, error) {
	started := time.Now()
	result := &Result{Stats: make(map[string]*Stats, len(AllTasks))}

	if c.config.DryRun {
		c.log.Warn("DRY RUN: data will be read and serialized, but nothing will be committed")
	}

	if c.config.Reset {
		if err := c.reset(ctx); err != nil {
			return result, err
		}
	}

	for _, task := range AllTasks {
		if c.config.skips(task) {
			c.log.Infof("Skipping task %q (excluded)", task)
			continue
		}

		stats, err := c.runTask(ctx, task)
		if stats != nil {
			result.Stats[task] = stats
		}
		if err != nil {
			return result, fmt.Errorf("task %q failed: %w", task, err)
		}
	}

	if c.config.Verify && !c.config.DryRun {
		warnings, err := c.verify(ctx, result)
		if err != nil {
			return result, err
		}
		result.Warnings = warnings
	}

	result.Duration = time.Since(started)
	return result, nil
}

func (c *Converter) runTask(ctx context.Context, task string) (*Stats, error) {
	switch task {
	case TaskExecutions:
		return newExecutionMigrator(c.mongo, c.pg, c.log, c.config).Migrate(ctx)
	case TaskSequences:
		return newSequenceMigrator(c.mongo, c.pg, c.log, c.config).Migrate(ctx)
	default:
		return nil, fmt.Errorf("unknown task %q", task)
	}
}

// reset clears both the target tables and the checkpoints, so the next run
// starts from scratch. It exists because COPY cannot express ON CONFLICT: a
// resumed run never overlaps what it already wrote, but an operator who wants to
// reload from the beginning needs the target emptied first, or the primary key
// on test_workflow_executions would reject the batch.
func (c *Converter) reset(ctx context.Context) error {
	if c.config.DryRun {
		c.log.Warn("dry run: skipping --reset truncation")
		return nil
	}

	c.log.Warn("--reset: truncating migrated tables and clearing checkpoints")

	if !c.config.skips(TaskExecutions) {
		if err := truncateExecutions(ctx, c.pg); err != nil {
			return err
		}
	}
	if !c.config.skips(TaskSequences) {
		if err := truncateSequences(ctx, c.pg); err != nil {
			return err
		}
	}
	return clearCheckpoints(ctx, c.pg)
}

// verify compares what is in Postgres against what Mongo holds. Mismatches are
// returned as warnings rather than errors so that the statistics still print and
// the operator can see the whole picture.
func (c *Converter) verify(ctx context.Context, result *Result) ([]string, error) {
	c.log.Info("Verifying migrated data")

	var warnings []string

	if stats, ok := result.Stats[TaskExecutions]; ok {
		pgCount, err := countExecutionRows(ctx, c.pg)
		if err != nil {
			return nil, fmt.Errorf("failed to count migrated executions: %w", err)
		}

		// Cumulative, not per-run: this compares the whole collection against the
		// whole table, so a resumed run has to discount what earlier runs declined
		// as well as its own. Using the per-run counters made a re-run with nothing
		// left to do report every previously declined document as a missing row.
		expected := stats.Total - stats.CumulativeFailed - stats.CumulativeSkipped
		c.log.Infof("Executions: %d in mongo, %d in postgres (%d expected)", stats.Total, pgCount, expected)
		if pgCount < expected {
			warnings = append(warnings, fmt.Sprintf(
				"executions: postgres holds %d rows but %d were expected (%d in mongo, %d failed, %d skipped)",
				pgCount, expected, stats.Total, stats.CumulativeFailed, stats.CumulativeSkipped))
		}

		missingStatus, err := countExecutionsMissingStatus(ctx, c.pg)
		if err != nil {
			return nil, fmt.Errorf("failed to count executions missing status: %w", err)
		}
		if missingStatus > 0 {
			// Either the result rows did not land or the denormalized column was
			// not written; both break the list and totals queries.
			warnings = append(warnings, fmt.Sprintf(
				"executions: %d rows have a NULL status, so their result data is incomplete", missingStatus))
		}
	}

	if stats, ok := result.Stats[TaskSequences]; ok {
		pgCount, err := countSequenceRows(ctx, c.pg)
		if err != nil {
			return nil, fmt.Errorf("failed to count migrated sequences: %w", err)
		}

		expected := stats.Total - stats.CumulativeFailed - stats.CumulativeSkipped
		c.log.Infof("Sequences: %d in mongo, %d in postgres (%d migratable)", stats.Total, pgCount, expected)
		if pgCount < expected {
			warnings = append(warnings, fmt.Sprintf(
				"sequences: postgres holds %d rows but %d were expected (%d in mongo, %d skipped as non-testworkflow)",
				pgCount, expected, stats.Total, stats.CumulativeSkipped))
		}
	}

	for _, w := range warnings {
		c.log.Warnf("VERIFY: %s", w)
	}
	if len(warnings) == 0 {
		c.log.Info("Verification passed")
	}

	return warnings, nil
}

// PrintSummary writes the closing report across all tasks.
func (r *Result) PrintSummary(log *zap.SugaredLogger) {
	rule := strings.Repeat("=", 70)

	log.Info(rule)
	log.Info("CONVERSION SUMMARY")
	log.Info(rule)
	for _, task := range AllTasks {
		stats, ok := r.Stats[task]
		if !ok {
			log.Infof("%-12s skipped", task)
			continue
		}
		log.Infof("%-12s processed=%d failed=%d skipped=%d of %d in %s",
			task, stats.Processed, stats.Failed, stats.Skipped, stats.Total,
			stats.Duration().Round(time.Millisecond))
	}
	log.Infof("Total duration: %s", r.Duration.Round(time.Millisecond))
	log.Info(rule)

	if len(r.Warnings) > 0 {
		log.Warnf("%d verification warning(s):", len(r.Warnings))
		for i, w := range r.Warnings {
			log.Warnf("  %d. %s", i+1, w)
		}
	}
}

// ErrIncomplete is returned to the caller when a run finished but did not
// migrate everything cleanly.
var ErrIncomplete = errors.New("conversion completed with failures")
