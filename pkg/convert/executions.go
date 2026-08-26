package convert

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// executionMigrator copies testworkflowresults documents into the seven
// test_workflow_* tables.
type executionMigrator struct {
	mongo  *mongo.Collection
	pg     *pgxpool.Pool
	log    *zap.SugaredLogger
	config Config

	// Counters carried over from a previous run's checkpoint. The Stats counters
	// describe only the current run, so these are added back when persisting a
	// checkpoint to keep the stored totals cumulative.
	priorProcessed int64
	priorFailed    int64
	priorSkipped   int64
}

func newExecutionMigrator(db *mongo.Database, pg *pgxpool.Pool, log *zap.SugaredLogger, cfg Config) *executionMigrator {
	return &executionMigrator{
		mongo:  db.Collection(executionsCollection),
		pg:     pg,
		log:    log,
		config: cfg,
	}
}

// executionBuffers holds one batch's worth of serialized rows, one buffer per
// target table. Rows are accumulated in memory rather than in temp files: a
// batch is bounded by Config.BatchSize, and keeping it in memory removes the
// create/sync/seek/cleanup dance plus the failure modes that come with it.
type executionBuffers struct {
	executions           bytes.Buffer
	signatures           bytes.Buffer
	results              bytes.Buffer
	outputs              bytes.Buffer
	reports              bytes.Buffer
	resourceAggregations bytes.Buffer
	workflows            bytes.Buffer
}

func (b *executionBuffers) reset() {
	b.executions.Reset()
	b.signatures.Reset()
	b.results.Reset()
	b.outputs.Reset()
	b.reports.Reset()
	b.resourceAggregations.Reset()
	b.workflows.Reset()
}

// Migrate walks the collection from the stored checkpoint and copies it across
// in batches. Each batch is committed together with its checkpoint, so an
// interrupted run resumes exactly where it stopped.
//
// The checkpoint's completed flag is deliberately not used to short-circuit: the
// stored position already makes a finished run a no-op, and honouring the flag
// would instead refuse executions that arrived after it was set, which is
// exactly what happens when an operator converts before the final cutover.
func (m *executionMigrator) Migrate(ctx context.Context) (*Stats, error) {
	stats := newStats(TaskExecutions)
	defer stats.finish()

	total, err := m.mongo.CountDocuments(ctx, bson.M{})
	if err != nil {
		return stats, fmt.Errorf("failed to count executions: %w", err)
	}
	stats.Total = total

	cp, err := loadCheckpoint(ctx, m.pg, TaskExecutions)
	if err != nil {
		return stats, err
	}

	m.priorProcessed, m.priorFailed, m.priorSkipped = cp.Processed, cp.Failed, cp.Skipped

	resume := &resumeInfo{Resumed: cp.LastMongoID != nil, AlreadyDone: cp.Processed}
	if resume.Resumed {
		m.log.Infof("Resuming executions after mongo _id %s (%d already migrated)",
			cp.LastMongoID.Hex(), cp.Processed)
	}
	m.log.Infof("Executions to migrate: %d total, batch size %d", total, m.config.BatchSize)

	if err := m.run(ctx, stats, cp); err != nil {
		return stats, err
	}

	if err := markCheckpointComplete(ctx, m.pg, cp); err != nil {
		return stats, err
	}

	stats.finish()
	stats.Print(m.log, resume)
	return stats, nil
}

func (m *executionMigrator) run(ctx context.Context, stats *Stats, cp *checkpoint) error {
	// Page on Mongo's own _id rather than on the execution's id field: _id is an
	// ObjectID, always present, always indexed, and orders consistently, which is
	// what makes the checkpoint a valid resume point.
	filter := bson.M{}
	if cp.LastMongoID != nil {
		filter["_id"] = bson.M{"$gt": *cp.LastMongoID}
	}

	cursor, err := m.mongo.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "_id", Value: 1}}).
			SetBatchSize(int32(m.config.ReadBatchSize)))
	if err != nil {
		return fmt.Errorf("failed to open executions cursor: %w", err)
	}
	defer cursor.Close(ctx) //nolint:errcheck // best effort on the read path

	var (
		buffers  executionBuffers
		counts   batchCounts
		inBatch  int
		batchEnd bson.ObjectID
		// pending records that batchEnd has moved past what the checkpoint holds.
		// It is not the same as inBatch > 0: a document that could not be migrated
		// advances the position without staging any rows, and its position still
		// has to be committed or the next run would read it again, fail on it
		// again, and never make progress past it.
		pending bool
		// Guards against duplicate ids inside a single batch, which would abort
		// the whole COPY on the primary key. Scoped per batch on purpose:
		// _id paging already rules out duplicates across batches, and a run-wide
		// set would grow without bound.
		seen = make(map[string]struct{}, m.config.BatchSize)
	)

	flush := func() error {
		if !pending {
			return nil
		}
		if err := m.commitBatch(ctx, &buffers, &counts, batchEnd, stats, cp); err != nil {
			return err
		}
		buffers.reset()
		counts = batchCounts{}
		inBatch = 0
		pending = false
		seen = make(map[string]struct{}, m.config.BatchSize)
		m.log.Infof("Executions: %d/%d migrated (%d batches, %d failed, %d skipped)",
			stats.Processed, stats.Total, stats.Batches, stats.Failed, stats.Skipped)
		return nil
	}

	for cursor.Next(ctx) {
		var raw struct {
			MongoID bson.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&raw); err != nil {
			return fmt.Errorf("failed to decode execution _id: %w", err)
		}

		// Decode into the same struct the API serves. The document is left in its
		// stored form -- notably its dot-escaped map keys are NOT unescaped here,
		// because the Postgres repository stores them escaped too, so a raw
		// passthrough is exactly right.
		var execution testkube.TestWorkflowExecution
		if err := cursor.Decode(&execution); err != nil {
			stats.addError("failed to decode execution %s: %v", raw.MongoID.Hex(), err)
			if !m.config.SkipErrors {
				return fmt.Errorf("failed to decode execution %s: %w", raw.MongoID.Hex(), err)
			}
			batchEnd, pending = raw.MongoID, true
			continue
		}

		if err := m.stage(&buffers, &counts, &execution, seen, stats); err != nil {
			if !m.config.SkipErrors {
				return err
			}
			batchEnd, pending = raw.MongoID, true
			continue
		}

		batchEnd, pending = raw.MongoID, true
		inBatch++

		if inBatch >= m.config.BatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("executions cursor failed: %w", err)
	}

	return flush()
}

// batchCounts tracks child-row totals for the batch currently being staged, so
// that a rolled-back batch does not inflate the reported stats.
type batchCounts struct {
	executions           int64
	signatures           int64
	results              int64
	outputs              int64
	reports              int64
	resourceAggregations int64
	workflows            int64
}

// stage serializes one execution into the batch buffers.
//
// Every row is written into per-execution scratch buffers first and only
// appended to the batch buffers once all of them succeeded. Without that, a
// failure partway through would leave a parent row in the executions buffer
// with no matching children, and the batch would import an execution whose
// result or signatures are silently missing.
func (m *executionMigrator) stage(dst *executionBuffers, counts *batchCounts,
	execution *testkube.TestWorkflowExecution, seen map[string]struct{}, stats *Stats) error {
	if err := validateExecution(execution); err != nil {
		stats.addError("%v", err)
		return err
	}

	if _, dup := seen[execution.Id]; dup {
		stats.addError("duplicate execution id %s within one batch", execution.Id)
		return fmt.Errorf("duplicate execution id %s within one batch", execution.Id)
	}

	// Legacy executions predate ScheduledAt; without this their scheduled_at
	// lands as NULL and renders as the Unix epoch, and durations derived from a
	// zero anchor wrap negative.
	repairExecutionTime(execution)

	var scratch executionBuffers
	staged := batchCounts{executions: 1}

	if err := writeExecutionRow(&scratch.executions, execution); err != nil {
		stats.addError("failed to serialize execution %s: %v", execution.Id, err)
		return err
	}

	sigOrder := int32(0)
	if err := writeSignatureRows(&scratch.signatures, execution.Id, execution.Signature, nil, &sigOrder); err != nil {
		stats.addError("failed to serialize signatures for %s: %v", execution.Id, err)
		return err
	}
	staged.signatures = int64(sigOrder)

	if execution.Result != nil {
		if err := writeResultRow(&scratch.results, execution.Id, execution.Result); err != nil {
			stats.addError("failed to serialize result for %s: %v", execution.Id, err)
			return err
		}
		staged.results = 1
	}

	for i := range execution.Output {
		if err := writeOutputRow(&scratch.outputs, execution.Id, &execution.Output[i], int32(i+1)); err != nil {
			stats.addError("failed to serialize output %d for %s: %v", i, execution.Id, err)
			return err
		}
		staged.outputs++
	}

	for i := range execution.Reports {
		if err := writeReportRow(&scratch.reports, execution.Id, &execution.Reports[i], int32(i+1)); err != nil {
			stats.addError("failed to serialize report %d for %s: %v", i, execution.Id, err)
			return err
		}
		staged.reports++
	}

	if execution.ResourceAggregations != nil {
		if err := writeResourceAggregationRow(&scratch.resourceAggregations, execution.Id,
			execution.ResourceAggregations); err != nil {
			stats.addError("failed to serialize resource aggregations for %s: %v", execution.Id, err)
			return err
		}
		staged.resourceAggregations = 1
	}

	if execution.Workflow != nil {
		if err := writeWorkflowRow(&scratch.workflows, execution.Id, workflowTypeWorkflow, execution.Workflow); err != nil {
			stats.addError("failed to serialize workflow for %s: %v", execution.Id, err)
			return err
		}
		staged.workflows++
	}

	if execution.ResolvedWorkflow != nil {
		if err := writeWorkflowRow(&scratch.workflows, execution.Id, workflowTypeResolvedWorkflow,
			execution.ResolvedWorkflow); err != nil {
			stats.addError("failed to serialize resolved workflow for %s: %v", execution.Id, err)
			return err
		}
		staged.workflows++
	}

	// All rows serialized -- promote the scratch buffers into the batch.
	dst.executions.Write(scratch.executions.Bytes())
	dst.signatures.Write(scratch.signatures.Bytes())
	dst.results.Write(scratch.results.Bytes())
	dst.outputs.Write(scratch.outputs.Bytes())
	dst.reports.Write(scratch.reports.Bytes())
	dst.resourceAggregations.Write(scratch.resourceAggregations.Bytes())
	dst.workflows.Write(scratch.workflows.Bytes())

	counts.executions += staged.executions
	counts.signatures += staged.signatures
	counts.results += staged.results
	counts.outputs += staged.outputs
	counts.reports += staged.reports
	counts.resourceAggregations += staged.resourceAggregations
	counts.workflows += staged.workflows

	seen[execution.Id] = struct{}{}
	return nil
}

// validateExecution rejects documents that cannot satisfy the target schema.
// escapeCopyValue maps the empty string to a NULL sentinel, so an empty value
// in a NOT NULL column would abort the entire batch rather than one row.
func validateExecution(execution *testkube.TestWorkflowExecution) error {
	if execution.Id == "" {
		return fmt.Errorf("execution %q has no id", execution.Name)
	}
	if execution.Name == "" {
		return fmt.Errorf("execution %s has no name (test_workflow_executions.name is NOT NULL)", execution.Id)
	}
	if execution.Workflow != nil && execution.Workflow.Name == "" {
		return fmt.Errorf("execution %s has a workflow snapshot with no name (test_workflows.name is NOT NULL)", execution.Id)
	}
	if execution.ResolvedWorkflow != nil && execution.ResolvedWorkflow.Name == "" {
		return fmt.Errorf("execution %s has a resolved workflow snapshot with no name (test_workflows.name is NOT NULL)", execution.Id)
	}
	return nil
}

// commitBatch imports one batch and advances the checkpoint in a single
// transaction. Executions are copied before their children so the foreign keys
// and the two denormalization triggers all resolve within the transaction.
func (m *executionMigrator) commitBatch(ctx context.Context, buffers *executionBuffers, counts *batchCounts,
	batchEnd bson.ObjectID, stats *Stats, cp *checkpoint) error {
	if m.config.DryRun {
		m.applyCounts(stats, counts)
		countBatch(stats, counts)
		m.log.Debugf("dry run: would commit %d executions", counts.executions)
		return nil
	}

	tx, err := m.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin batch transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	copies := []struct {
		table   string
		buf     *bytes.Buffer
		columns []string
	}{
		{"test_workflow_executions", &buffers.executions, executionColumns},
		{"test_workflow_signatures", &buffers.signatures, signatureColumns},
		{"test_workflow_results", &buffers.results, resultColumns},
		{"test_workflow_outputs", &buffers.outputs, outputColumns},
		{"test_workflow_reports", &buffers.reports, reportColumns},
		{"test_workflow_resource_aggregations", &buffers.resourceAggregations, resourceAggregationColumns},
		{"test_workflows", &buffers.workflows, workflowColumns},
	}

	for _, c := range copies {
		if c.buf.Len() == 0 {
			continue
		}
		if _, err := copyFrom(ctx, tx, c.table, bytes.NewReader(c.buf.Bytes()), c.columns...); err != nil {
			return err
		}
	}

	// Advance the checkpoint inside the same transaction. This is what makes the
	// migration exactly-once: progress can never be durable without its data,
	// and the data can never be durable without its progress.
	next := batchEnd
	cp.LastMongoID = &next
	// The stats counters cover this run only, while the checkpoint is cumulative
	// across every run, so the totals carried over from the previous run have to
	// be added back or a resume would make the stored counts go backwards.
	cp.Processed = m.priorProcessed + stats.Processed + counts.executions
	cp.Failed = m.priorFailed + stats.Failed
	cp.Skipped = m.priorSkipped + stats.Skipped
	if err := saveCheckpoint(ctx, tx, cp); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	m.applyCounts(stats, counts)
	countBatch(stats, counts)
	return nil
}

// countBatch records a committed batch, ignoring one that carried no rows. A
// trailing run of documents that could not be migrated still has to commit, to
// move the checkpoint past them, but reporting that as a batch would overstate
// the work done.
func countBatch(stats *Stats, counts *batchCounts) {
	if counts.executions > 0 {
		stats.Batches++
	}
}

func (m *executionMigrator) applyCounts(stats *Stats, counts *batchCounts) {
	stats.Processed += counts.executions
	stats.Signatures += counts.signatures
	stats.Results += counts.results
	stats.Outputs += counts.outputs
	stats.Reports += counts.reports
	stats.ResourceAggregations += counts.resourceAggregations
	stats.Workflows += counts.workflows
}

// truncateExecutions removes every migrated execution row. Children go with it
// through ON DELETE CASCADE.
func truncateExecutions(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `TRUNCATE test_workflow_executions CASCADE`); err != nil {
		return fmt.Errorf("failed to truncate executions: %w", err)
	}
	return nil
}

// countExecutionRows is used by the verification pass.
func countExecutionRows(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	var n int64
	if err := db.QueryRow(ctx, `SELECT count(*) FROM test_workflow_executions`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// countExecutionsMissingStatus reports executions whose denormalized status did
// not land, which would mean the result rows or the status column were lost.
func countExecutionsMissingStatus(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	var n int64
	err := db.QueryRow(ctx,
		`SELECT count(*) FROM test_workflow_executions WHERE status IS NULL`).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}
