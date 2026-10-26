package scheduling

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
)

func dispatchRow() sqlc.TestWorkflowExecution {
	return sqlc.TestWorkflowExecution{
		ID:              "exec-1",
		Name:            "wf-a-7",
		GroupID:         pgtype.Text{String: "group-1", Valid: true},
		RunnerID:        pgtype.Text{String: "standalone", Valid: true},
		Number:          pgtype.Int4{Int32: 7, Valid: true},
		ScheduledAt:     pgtype.Timestamptz{Time: time.Unix(1735689600, 0), Valid: true},
		StatusAt:        pgtype.Timestamptz{Time: time.Unix(1735689601, 0), Valid: true},
		DisableWebhooks: pgtype.Bool{Bool: true, Valid: true},
		Tags:            map[string]string{"env": "prod"},
		WorkflowName:    pgtype.Text{String: "wf-a", Valid: true},
		Status:          pgtype.Text{String: string(testkube.ASSIGNED_TestWorkflowStatus), Valid: true},
		LineageBaseID:   pgtype.Text{String: "exec-0", Valid: true},
		LineageRootID:   pgtype.Text{String: "exec-0", Valid: true},
		LineageAttempt:  pgtype.Int4{Int32: 2, Valid: true},
		RunningContext: &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{ExecutionPath: "p1/p2"},
		},
	}
}

// The dispatch projection has to carry everything the runner is sent, from one
// table, without reading the workflow spec.
func TestMapPgTestWorkflowExecutionForDispatch(t *testing.T) {
	out := mapPgTestWorkflowExecutionForDispatch(dispatchRow())

	assert.Equal(t, "exec-1", out.Id)
	assert.Equal(t, "group-1", out.GroupId)
	assert.Equal(t, "wf-a-7", out.Name)
	assert.Equal(t, int32(7), out.Number)
	assert.True(t, out.DisableWebhooks)
	assert.Equal(t, map[string]string{"env": "prod"}, out.Tags)
	require.NotNil(t, out.RunningContext)
	require.NotNil(t, out.RunningContext.Actor)
	assert.Equal(t, "p1/p2", out.RunningContext.Actor.ExecutionPath)

	// The denormalised columns stand in for the joins the dispatch query no
	// longer performs.
	require.NotNil(t, out.Workflow, "the runner needs the workflow name to log against")
	assert.Equal(t, "wf-a", out.Workflow.Name)
	require.NotNil(t, out.Result)
	require.NotNil(t, out.Result.Status)
	assert.Equal(t, testkube.ASSIGNED_TestWorkflowStatus, *out.Result.Status,
		"the handler switches on this, so e.status has to be mapped through")

	// Lineage is carried, or a workflow written against its own previous run
	// silently loses the reference.
	require.NotNil(t, out.Lineage)
	assert.Equal(t, "exec-0", out.Lineage.BaseId)
	assert.Equal(t, "exec-0", out.Lineage.RootId)
	assert.Equal(t, int32(2), out.Lineage.Attempt)
}

// Everything that used to cost a query per row must stay unset. If any of these
// becomes populated again, the six per-row reads have come back with it.
func TestMapPgTestWorkflowExecutionForDispatch_HydratesNothing(t *testing.T) {
	out := mapPgTestWorkflowExecutionForDispatch(dispatchRow())

	assert.Nil(t, out.ResolvedWorkflow, "the runner resolves the workflow itself")
	assert.Nil(t, out.Signature)
	assert.Nil(t, out.Output)
	assert.Nil(t, out.Reports)
	assert.Nil(t, out.ResourceAggregations)
	require.NotNil(t, out.Workflow)
	assert.Nil(t, out.Workflow.Spec, "the workflow spec is the expensive part; it must not be read here")
}

// A row written before the workflow_name column was denormalised carries none,
// and nil has to stay nil rather than becoming an empty workflow.
func TestMapPgTestWorkflowExecutionForDispatch_LeavesWorkflowUnsetWithoutAName(t *testing.T) {
	row := dispatchRow()
	row.WorkflowName = pgtype.Text{}

	out := mapPgTestWorkflowExecutionForDispatch(row)

	assert.Nil(t, out.Workflow)
}

// BenchmarkMapExecutionForDispatch and BenchmarkMapExecutionFull measure mapper
// overhead only.
//
// They do NOT measure the jsonb decode: sqlc.TestWorkflow.Spec is already a
// decoded *testkube.TestWorkflowSpec by the time a mapper sees it, because pgx
// decodes it during the scan. What the full mapper pays on top of the dispatch
// one is the signature/output/report tree building, plus the six queries whose
// scans do that decoding - and those only show up in the DB-backed benchmark.
func BenchmarkMapExecutionForDispatch(b *testing.B) {
	row := dispatchRow()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mapPgTestWorkflowExecutionForDispatch(row)
	}
}

func BenchmarkMapExecutionFull(b *testing.B) {
	row := dispatchRow()
	result := sqlc.TestWorkflowResult{Status: row.Status}
	workflow := sqlc.TestWorkflow{
		Name: pgtype.Text{String: "wf-a", Valid: true},
		Spec: &testkube.TestWorkflowSpec{},
	}
	signatures := make([]sqlc.TestWorkflowSignature, 0, 32)
	for i := 0; i < 32; i++ {
		var id pgtype.UUID
		id.Bytes[0] = byte(i + 1)
		id.Valid = true
		signatures = append(signatures, sqlc.TestWorkflowSignature{
			ID:       id,
			SigOrder: int32(i),
			Ref:      pgtype.Text{String: "step", Valid: true},
			Name:     pgtype.Text{String: "step", Valid: true},
		})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mapPgTestWorkflowExecution(row, result, workflow, workflow, signatures, nil, nil, sqlc.TestWorkflowResourceAggregation{})
	}
}
