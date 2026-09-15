package scheduling

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
)

func lineageRow() sqlc.TestWorkflowExecution {
	return sqlc.TestWorkflowExecution{
		ID:             "exec-3",
		Name:           "wf-3",
		LineageBaseID:  pgtype.Text{String: "exec-2", Valid: true},
		LineageRootID:  pgtype.Text{String: "exec-1", Valid: true},
		LineageAttempt: pgtype.Int4{Int32: 3, Valid: true},
	}
}

// Both mappers feed createExecutionStart, which projects Lineage onto the
// ExecutionStart. Dropping it here sends the pod no lineage, so
// execution("rerun") stops resolving for every rerun on the Postgres backend -
// and nothing reports it, which is why both are pinned rather than one.
func TestMapPgTestWorkflowExecutionCarriesLineage(t *testing.T) {
	partial := mapPgTestWorkflowExecutionPartial(lineageRow(), sqlc.TestWorkflowResult{})
	require.NotNil(t, partial.Lineage, "partial mapper dropped lineage")
	assert.Equal(t, "exec-2", partial.Lineage.BaseId)
	assert.Equal(t, "exec-1", partial.Lineage.RootId)
	assert.Equal(t, int32(3), partial.Lineage.Attempt)

	full := mapPgTestWorkflowExecution(lineageRow(), sqlc.TestWorkflowResult{}, sqlc.TestWorkflow{},
		sqlc.TestWorkflow{}, nil, nil, nil, sqlc.TestWorkflowResourceAggregation{})
	require.NotNil(t, full.Lineage, "full mapper dropped lineage")
	assert.Equal(t, "exec-2", full.Lineage.BaseId)
	assert.Equal(t, "exec-1", full.Lineage.RootId)
	assert.Equal(t, int32(3), full.Lineage.Attempt)
}

// A row written before lineage existed carries three NULLs and must come back
// nil, so EffectiveLineage() supplies the original-run default rather than the
// row claiming attempt zero.
func TestMapPgTestWorkflowExecutionWithoutLineage(t *testing.T) {
	row := sqlc.TestWorkflowExecution{ID: "exec-1", Name: "wf-1"}

	partial := mapPgTestWorkflowExecutionPartial(row, sqlc.TestWorkflowResult{})
	assert.Nil(t, partial.Lineage)
	assert.Equal(t, "exec-1", partial.EffectiveLineage().RootId)
	assert.Equal(t, int32(1), partial.EffectiveLineage().Attempt)

	full := mapPgTestWorkflowExecution(row, sqlc.TestWorkflowResult{}, sqlc.TestWorkflow{},
		sqlc.TestWorkflow{}, nil, nil, nil, sqlc.TestWorkflowResourceAggregation{})
	assert.Nil(t, full.Lineage)
}

// An original run records a root and an attempt but no base; that row is still
// lineage and must not read back as nil.
func TestMapPgTestWorkflowExecutionLineageForAnOriginalRun(t *testing.T) {
	lineage := mapPgTestWorkflowExecutionLineage(sqlc.TestWorkflowExecution{
		ID:             "exec-1",
		LineageRootID:  pgtype.Text{String: "exec-1", Valid: true},
		LineageAttempt: pgtype.Int4{Int32: 1, Valid: true},
	})

	require.NotNil(t, lineage)
	assert.Empty(t, lineage.BaseId)
	assert.Equal(t, "exec-1", lineage.RootId)
	assert.Equal(t, int32(1), lineage.Attempt)
}
