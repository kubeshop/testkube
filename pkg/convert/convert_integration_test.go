package convert

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	sequencemongo "github.com/kubeshop/testkube/pkg/repository/sequence/mongo"
	testworkflowmongo "github.com/kubeshop/testkube/pkg/repository/testworkflow/mongo"
	testworkflowpostgres "github.com/kubeshop/testkube/pkg/repository/testworkflow/postgres"
	testmongo "github.com/kubeshop/testkube/pkg/test/mongo"
	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

// fixture builds the databases and returns everything the tests need.
type fixture struct {
	mongoDB   *mongo.Database
	pg        *pgxpool.Pool
	mongoRepo *testworkflowmongo.MongoRepository
	pgRepo    *testworkflowpostgres.PostgresRepository
	log       *zap.SugaredLogger
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	mongoDB, _ := testmongo.PrepareMongoTestDatabase(t, "convert")
	pgDB, _ := testpostgres.PreparePostgresTestDatabase(t, "convert")

	return &fixture{
		mongoDB:   mongoDB,
		pg:        pgDB.Pool,
		mongoRepo: testworkflowmongo.NewMongoRepository(mongoDB, false),
		pgRepo:    testworkflowpostgres.NewPostgresRepository(pgDB.Pool),
		log:       zap.NewNop().Sugar(),
	}
}

// convert runs a migration with verification forced on. Config.Verify is a bare
// bool, so a test that built its Config literally would otherwise silently get
// Verify=false and every assertion on Result.Warnings would pass vacuously.
func (f *fixture) convert(t *testing.T, cfg Config) *Result {
	t.Helper()

	cfg.Verify = true
	result, err := New(f.mongoDB, f.pg, f.log, cfg).Run(context.Background())
	require.NoError(t, err)
	return result
}

// buildExecution produces an execution exercising every table the migrator
// writes: nested signatures, a result, outputs, reports, resource aggregations,
// and both workflow snapshots.
func buildExecution(i int) testkube.TestWorkflowExecution {
	id := fmt.Sprintf("exec-%03d", i)
	name := fmt.Sprintf("wf-run-%03d", i)
	scheduled := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute)
	status := testkube.PASSED_TestWorkflowStatus
	if i%3 == 0 {
		status = testkube.FAILED_TestWorkflowStatus
	}

	return testkube.TestWorkflowExecution{
		Id:                        id,
		Name:                      name,
		Namespace:                 "testkube",
		Number:                    int32(i),
		GroupId:                   fmt.Sprintf("group-%d", i%3),
		RunnerId:                  "runner-1",
		TestWorkflowExecutionName: "scheduled",
		DisableWebhooks:           i%2 == 0,
		ScheduledAt:               scheduled,
		AssignedAt:                scheduled.Add(time.Second),
		StatusAt:                  scheduled.Add(30 * time.Second),
		// Dotted keys are stored escaped by both backends; the migrator must not
		// re-encode or decode them.
		Tags: map[string]string{"app.kubernetes.io/name": "testkube", "tier": "smoke"},
		ConfigParams: map[string]testkube.TestWorkflowExecutionConfigValue{
			"short": {Value: "v"},
			// Longer than configParamSizeLimit, so this catches any read path
			// that truncates on the way through.
			"long":      {Value: strings.Repeat("x", 250)},
			"sensitive": {Value: "super-secret", Sensitive: true},
		},
		Signature: []testkube.TestWorkflowSignature{{
			Ref:  "root",
			Name: "Root",
			Children: []testkube.TestWorkflowSignature{
				{
					Ref:      "child-a",
					Name:     "Child A",
					Optional: true,
					Children: []testkube.TestWorkflowSignature{
						{Ref: "grandchild", Name: "Grandchild", Negative: true},
					},
				},
				{Ref: "child-b", Name: "Child B", Category: "setup"},
			},
		}},
		Result: &testkube.TestWorkflowResult{
			Status:          &status,
			PredictedStatus: &status,
			QueuedAt:        scheduled,
			StartedAt:       scheduled.Add(time.Second),
			FinishedAt:      scheduled.Add(31 * time.Second),
			Duration:        "30s",
			TotalDuration:   "31s",
			DurationMs:      30000,
			TotalDurationMs: 31000,
			Steps: map[string]testkube.TestWorkflowStepResult{
				"root": {Status: stepStatusPtr(testkube.PASSED_TestWorkflowStepStatus)},
			},
		},
		Output: []testkube.TestWorkflowOutput{{
			Ref:   "root",
			Name:  "artifacts",
			Value: map[string]interface{}{"path": "/data/report.xml"},
		}},
		Reports: []testkube.TestWorkflowReport{{
			Ref:  "root",
			Kind: "junit",
			File: "report.xml",
			Summary: &testkube.TestWorkflowReportSummary{
				Tests:  10,
				Passed: 9,
				Failed: 1,
			},
		}},
		ResourceAggregations: &testkube.TestWorkflowExecutionResourceAggregationsReport{},
		Workflow: &testkube.TestWorkflow{
			Name:        fmt.Sprintf("workflow-%03d", i),
			Namespace:   "testkube",
			Description: "a migrated workflow",
			Labels:      map[string]string{"app.kubernetes.io/part-of": "testkube"},
			Created:     scheduled.Add(-time.Hour),
		},
		ResolvedWorkflow: &testkube.TestWorkflow{
			Name:      fmt.Sprintf("workflow-%03d", i),
			Namespace: "testkube",
			// A non-nil Spec.Config is what makes both repositories run
			// populateConfigParams on read, so this is what exercises the
			// truncation and sensitive-blanking paths that the migrator must not
			// bake into the stored rows.
			Spec: &testkube.TestWorkflowSpec{
				Config: map[string]testkube.TestWorkflowParameterSchema{
					"short":     {Default_: &testkube.BoxedString{Value: "d"}},
					"long":      {},
					"sensitive": {Sensitive: true},
				},
			},
		},
	}
}

func stepStatusPtr(s testkube.TestWorkflowStepStatus) *testkube.TestWorkflowStepStatus { return &s }

// seedExecutions writes executions through the Mongo repository, so the source
// documents are byte-for-byte what a real installation holds -- dot escaping
// included.
func seedExecutions(t *testing.T, f *fixture, count int) []testkube.TestWorkflowExecution {
	t.Helper()

	ctx := context.Background()
	seeded := make([]testkube.TestWorkflowExecution, 0, count)
	for i := 1; i <= count; i++ {
		execution := buildExecution(i)
		require.NoError(t, f.mongoRepo.Insert(ctx, execution))
		seeded = append(seeded, execution)
	}
	return seeded
}

func TestConvertExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seeded := seedExecutions(t, f, 50)

	result := f.convert(t, Config{BatchSize: 7})

	require.Empty(t, result.Warnings, "verification must pass")
	stats := result.Stats[TaskExecutions]
	require.NotNil(t, stats)
	assert.Equal(t, int64(50), stats.Total)
	assert.Equal(t, int64(50), stats.Processed)
	assert.Zero(t, stats.Failed)
	// 4 signature nodes and 2 workflow snapshots per execution.
	assert.Equal(t, int64(200), stats.Signatures)
	assert.Equal(t, int64(50), stats.Results)
	assert.Equal(t, int64(50), stats.Outputs)
	assert.Equal(t, int64(50), stats.Reports)
	assert.Equal(t, int64(100), stats.Workflows)

	// The real assertion: reading an execution back through the Postgres
	// repository must give the same value the Mongo repository serves. That is
	// what proves the hand-written COPY rows are indistinguishable from what
	// PostgresRepository.Insert would have written.
	for _, seed := range seeded {
		fromMongo, err := f.mongoRepo.Get(ctx, seed.Id)
		require.NoError(t, err)

		fromPostgres, err := f.pgRepo.Get(ctx, seed.Id)
		require.NoError(t, err, "execution %s must exist in postgres", seed.Id)

		assert.Equal(t, fromMongo, fromPostgres, "execution %s must round-trip identically", seed.Id)
	}
}

// Dotted map keys and oversized or sensitive config params are the values most
// likely to be mangled in transit, so they get their own assertions rather than
// relying on the bulk comparison above.
func TestConvertPreservesAwkwardValues_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seeded := seedExecutions(t, f, 1)
	f.convert(t, Config{})

	got, err := f.pgRepo.Get(ctx, seeded[0].Id)
	require.NoError(t, err)

	assert.Equal(t, "testkube", got.Tags["app.kubernetes.io/name"],
		"a dotted tag key must be readable after migrating")
	assert.Equal(t, "testkube", got.Workflow.Labels["app.kubernetes.io/part-of"],
		"a dotted label key must be readable after migrating")

	// Both read paths truncate long config values and blank sensitive ones, so
	// the migrator decodes raw documents rather than going through Get -- storing
	// an already-truncated value would make the loss permanent.
	fromMongo, err := f.mongoRepo.Get(ctx, seeded[0].Id)
	require.NoError(t, err)
	assert.Equal(t, fromMongo.ConfigParams, got.ConfigParams,
		"config params must be reported identically by both backends")

	// The derived view truncates, which is expected...
	assert.True(t, got.ConfigParams["long"].Truncated,
		"a long config value must be reported as truncated")
	assert.Len(t, got.ConfigParams["long"].Value, 100)
	assert.True(t, got.ConfigParams["sensitive"].Sensitive)
	assert.Empty(t, got.ConfigParams["sensitive"].Value,
		"a sensitive config value must not be served")

	// ...but the stored row must still hold the full value, proving the migrator
	// copied the raw document and not the lossy read-path projection.
	var stored string
	require.NoError(t, f.pg.QueryRow(ctx,
		`SELECT config_params->'long'->>'value' FROM test_workflow_executions WHERE id = $1`,
		seeded[0].Id).Scan(&stored))
	assert.Len(t, stored, 250, "the stored config value must not be truncated")
}

// The denormalized columns back the list and totals queries, so a NULL in
// either would make migrated executions invisible to the UI.
func TestConvertPopulatesDenormalizedColumns_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seedExecutions(t, f, 10)
	f.convert(t, Config{})

	var nullStatus, nullWorkflowName int64
	require.NoError(t, f.pg.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE status IS NULL),
		        count(*) FILTER (WHERE workflow_name IS NULL)
		   FROM test_workflow_executions`).Scan(&nullStatus, &nullWorkflowName))

	assert.Zero(t, nullStatus, "every migrated execution must carry its status")
	assert.Zero(t, nullWorkflowName, "every migrated execution must carry its workflow name")

	// And they must agree with the child tables they mirror.
	var mismatched int64
	require.NoError(t, f.pg.QueryRow(ctx, `
		SELECT count(*)
		  FROM test_workflow_executions e
		  JOIN test_workflow_results r ON r.execution_id = e.id
		  JOIN test_workflows w ON w.execution_id = e.id AND w.workflow_type = 'workflow'
		 WHERE e.status IS DISTINCT FROM r.status
		    OR e.workflow_name IS DISTINCT FROM w.name`).Scan(&mismatched))
	assert.Zero(t, mismatched, "denormalized columns must match their source rows")
}

func TestConvertSequences_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)

	sequences := f.mongoDB.Collection(sequencemongo.CollectionSequences)
	_, err := sequences.InsertMany(ctx, []interface{}{
		bson.M{"_id": "tw-workflow-a", "number": 17, "executionType": "tw"},
		bson.M{"_id": "tw-workflow-b", "number": 3, "executionType": "tw"},
		// Legacy counters for Tests and TestSuites have no Postgres consumer, and
		// the target keyspace ignores the execution type, so migrating them would
		// collide with the tw- entries.
		bson.M{"_id": "t-legacy-test", "number": 99, "executionType": "t"},
		bson.M{"_id": "ts-legacy-suite", "number": 42, "executionType": "ts"},
	})
	require.NoError(t, err)

	result := f.convert(t, Config{Skip: []string{TaskExecutions}})

	stats := result.Stats[TaskSequences]
	require.NotNil(t, stats)
	assert.Equal(t, int64(4), stats.Total)
	assert.Equal(t, int64(2), stats.Processed, "only test workflow counters are migrated")
	assert.Equal(t, int64(2), stats.Skipped, "legacy counters are skipped, not failed")
	assert.Zero(t, stats.Failed)

	assertSequence(t, f.pg, "workflow-a", 17)
	assertSequence(t, f.pg, "workflow-b", 3)

	var legacy int64
	require.NoError(t, f.pg.QueryRow(ctx,
		`SELECT count(*) FROM execution_sequences WHERE name IN ('legacy-test', 'ts-legacy-suite')`).Scan(&legacy))
	assert.Zero(t, legacy, "legacy counters must not reach postgres")
}

// A migrated counter must never move backwards, or executions scheduled after
// the cutover would reuse the numbers of migrated ones.
func TestConvertSequencesNeverGoBackwards_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)

	sequences := f.mongoDB.Collection(sequencemongo.CollectionSequences)
	_, err := sequences.InsertOne(ctx, bson.M{"_id": "tw-workflow-a", "number": 17, "executionType": "tw"})
	require.NoError(t, err)

	f.convert(t, Config{Skip: []string{TaskExecutions}})
	assertSequence(t, f.pg, "workflow-a", 17)

	// Simulate the API server having advanced the counter after the migration,
	// then re-run: the higher value must survive.
	_, err = f.pg.Exec(ctx,
		`UPDATE execution_sequences SET number = 25 WHERE name = 'workflow-a'`)
	require.NoError(t, err)

	f.convert(t, Config{Skip: []string{TaskExecutions}})
	assertSequence(t, f.pg, "workflow-a", 25,
		"re-running must not lower a counter the control plane has advanced")
}

// Re-running a completed migration must be a no-op rather than a duplicate-key
// failure, because a Kubernetes Job can be restarted at any time.
func TestConvertIsIdempotent_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seedExecutions(t, f, 20)

	first := f.convert(t, Config{BatchSize: 6})
	assert.Equal(t, int64(20), first.Stats[TaskExecutions].Processed)

	before := countRows(t, f.pg)

	second := f.convert(t, Config{BatchSize: 6})
	require.Empty(t, second.Warnings)
	assert.Zero(t, second.Stats[TaskExecutions].Processed,
		"a completed run must re-copy nothing")
	assert.Zero(t, second.Stats[TaskExecutions].Failed)

	assert.Equal(t, before, countRows(t, f.pg), "row counts must be unchanged")

	var distinct, total int64
	require.NoError(t, f.pg.QueryRow(ctx,
		`SELECT count(DISTINCT id), count(*) FROM test_workflow_executions`).Scan(&distinct, &total))
	assert.Equal(t, distinct, total, "no duplicate executions")
}

// Interrupting a run must lose at most the batch in flight, and resuming must
// land every remaining execution exactly once.
func TestConvertResumesFromCheckpoint_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seedExecutions(t, f, 30)

	// Cancel once two batches of five have been committed, standing in for a
	// killed pod.
	cancelCtx, cancel := context.WithCancel(ctx)
	converter := New(f.mongoDB, f.pg, f.log, Config{BatchSize: 5, Verify: false})

	go func() {
		for {
			if n := countExecutions(t, f.pg); n >= 10 {
				cancel()
				return
			}
			select {
			case <-cancelCtx.Done():
				return
			case <-time.After(5 * time.Millisecond):
			}
		}
	}()

	// The interrupted run is expected to fail; what matters is the state it left.
	_, _ = converter.Run(cancelCtx)

	partial := countExecutions(t, f.pg)
	require.Greater(t, partial, int64(0), "the interrupted run must have committed something")
	require.Less(t, partial, int64(30), "the interrupted run must not have finished")

	// Resume. Note this is NOT a --reset: the checkpoint drives it.
	result := f.convert(t, Config{BatchSize: 5})
	require.Empty(t, result.Warnings)

	assert.Equal(t, int64(30), countExecutions(t, f.pg),
		"resuming must complete the migration")

	var distinct, total int64
	require.NoError(t, f.pg.QueryRow(ctx,
		`SELECT count(DISTINCT id), count(*) FROM test_workflow_executions`).Scan(&distinct, &total))
	assert.Equal(t, int64(30), distinct)
	assert.Equal(t, distinct, total, "resuming must not duplicate the batch in flight")

	// The checkpoint counter is cumulative across runs, so it must reflect
	// everything migrated rather than only what the resuming run did.
	var checkpointed int64
	require.NoError(t, f.pg.QueryRow(ctx,
		`SELECT processed_count FROM convert_checkpoints WHERE task = $1`,
		TaskExecutions).Scan(&checkpointed))
	assert.Equal(t, int64(30), checkpointed,
		"the checkpoint must not lose the count from the interrupted run")
}

func TestConvertReset_Integration(t *testing.T) {
	test.IntegrationTest(t)

	f := newFixture(t)
	seedExecutions(t, f, 10)

	f.convert(t, Config{})
	require.Equal(t, int64(10), countExecutions(t, f.pg))

	// --reset must clear the target and the checkpoint, then reload everything.
	result := f.convert(t, Config{Reset: true})
	require.Empty(t, result.Warnings)
	assert.Equal(t, int64(10), result.Stats[TaskExecutions].Processed,
		"reset must re-migrate from the beginning")
	assert.Equal(t, int64(10), countExecutions(t, f.pg))
}

// A dry run must read and serialize everything without writing, which is what
// makes it useful as a pre-flight check.
func TestConvertDryRun_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seedExecutions(t, f, 10)

	result := f.convert(t, Config{DryRun: true})

	assert.Equal(t, int64(10), result.Stats[TaskExecutions].Processed,
		"a dry run must still serialize every execution")
	assert.Zero(t, result.Stats[TaskExecutions].Failed)
	assert.Zero(t, countExecutions(t, f.pg), "a dry run must write nothing")

	var checkpoints int64
	require.NoError(t, f.pg.QueryRow(ctx, `SELECT count(*) FROM convert_checkpoints`).Scan(&checkpoints))
	assert.Zero(t, checkpoints, "a dry run must not record progress")
}

// An execution that cannot satisfy the target schema must be reported and, with
// --skip-errors, must not take the rest of its batch down with it.
func TestConvertSkipsInvalidExecutions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seedExecutions(t, f, 5)

	// name is NOT NULL in Postgres, so this document is unmigratable. It is
	// inserted directly to bypass the repository.
	_, err := f.mongoDB.Collection(testworkflowmongo.CollectionName).
		InsertOne(ctx, bson.M{"id": "exec-broken", "name": ""})
	require.NoError(t, err)

	result, err := New(f.mongoDB, f.pg, f.log, Config{SkipErrors: true, Verify: true}).Run(ctx)
	require.NoError(t, err)

	stats := result.Stats[TaskExecutions]
	assert.Equal(t, int64(6), stats.Total)
	assert.Equal(t, int64(5), stats.Processed, "the valid executions must still land")
	assert.Equal(t, int64(1), stats.Failed)
	require.Len(t, stats.Errors, 1)
	assert.Contains(t, stats.Errors[0], "has no name")

	// Verification must discount the failure rather than flag the shortfall,
	// otherwise every skipped document would also raise a spurious warning.
	assert.Empty(t, result.Warnings, "a known failure is not a verification mismatch")
	assert.True(t, result.Failed(), "but the run must still report failure")
	assert.Equal(t, int64(5), countExecutions(t, f.pg))
}

// Without --skip-errors the same document must abort the run, so an operator
// cannot silently end up with a partial migration.
func TestConvertFailsFastByDefault_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	f := newFixture(t)
	seedExecutions(t, f, 5)

	_, err := f.mongoDB.Collection(testworkflowmongo.CollectionName).
		InsertOne(ctx, bson.M{"id": "exec-broken", "name": ""})
	require.NoError(t, err)

	_, err = New(f.mongoDB, f.pg, f.log, Config{}).Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no name")
}

func assertSequence(t *testing.T, pg *pgxpool.Pool, name string, want int32, msgAndArgs ...interface{}) {
	t.Helper()

	var got int32
	err := pg.QueryRow(context.Background(),
		`SELECT number FROM execution_sequences WHERE name = $1 AND organization_id = '' AND environment_id = ''`,
		name).Scan(&got)
	require.NoError(t, err, "sequence %q must exist", name)
	assert.Equal(t, want, got, msgAndArgs...)
}

func countExecutions(t *testing.T, pg *pgxpool.Pool) int64 {
	t.Helper()

	var n int64
	require.NoError(t, pg.QueryRow(context.Background(),
		`SELECT count(*) FROM test_workflow_executions`).Scan(&n))
	return n
}

// countRows snapshots every migrated table, so an idempotency check catches a
// duplicate in any of them rather than only in the parent.
func countRows(t *testing.T, pg *pgxpool.Pool) map[string]int64 {
	t.Helper()

	tables := []string{
		"test_workflow_executions",
		"test_workflow_signatures",
		"test_workflow_results",
		"test_workflow_outputs",
		"test_workflow_reports",
		"test_workflow_resource_aggregations",
		"test_workflows",
		"execution_sequences",
	}

	out := make(map[string]int64, len(tables))
	for _, table := range tables {
		var n int64
		require.NoError(t, pg.QueryRow(context.Background(),
			fmt.Sprintf(`SELECT count(*) FROM %s`, table)).Scan(&n))
		out[table] = n
	}
	return out
}
