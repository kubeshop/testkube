package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// A row written before lineage existed has all three columns NULL and must come
// back as nil, not as a zero-valued record. The default belongs to
// EffectiveLineage; synthesizing one here would report an original run as
// attempt zero, with no way for the caller to tell that apart from a real value.
func TestLineageFromRowWithoutColumns(t *testing.T) {
	assert.Nil(t, lineageFromRow(pgtype.Text{}, pgtype.Text{}, pgtype.Int4{}))
}

func TestLineageFromRow(t *testing.T) {
	lineage := lineageFromRow(
		pgtype.Text{String: "exec-2", Valid: true},
		pgtype.Text{String: "exec-1", Valid: true},
		pgtype.Int4{Int32: 3, Valid: true},
	)

	require.NotNil(t, lineage)
	assert.Equal(t, "exec-2", lineage.BaseId)
	assert.Equal(t, "exec-1", lineage.RootId)
	assert.Equal(t, int32(3), lineage.Attempt)
}

// An original run has a root and an attempt but no base, and that row is still
// lineage - it must not read back as nil.
func TestLineageFromRowForAnOriginalRun(t *testing.T) {
	lineage := lineageFromRow(
		pgtype.Text{},
		pgtype.Text{String: "exec-1", Valid: true},
		pgtype.Int4{Int32: 1, Valid: true},
	)

	require.NotNil(t, lineage)
	assert.Empty(t, lineage.BaseId)
	assert.Equal(t, "exec-1", lineage.RootId)
	assert.Equal(t, int32(1), lineage.Attempt)
}

// An original run's empty base must reach the column as SQL NULL rather than as
// an empty string: the partial index covers "has a lineage", and an empty string
// would make every original run a row the index has to carry. It is also the
// scalar analogue of the JSONB "null"-versus-NULL trap toJSONB exists to avoid.
func TestLineageColumnsForAnOriginalRun(t *testing.T) {
	lineage := &testkube.TestWorkflowExecutionLineage{RootId: "exec-1", Attempt: 1}

	assert.False(t, toPgText(lineageBaseID(lineage)).Valid)
	assert.True(t, toPgText(lineageRootID(lineage)).Valid)
	assert.Equal(t, pgtype.Int4{Int32: 1, Valid: true}, lineageAttempt(lineage))
}

// No lineage at all writes three NULLs, so the row is indistinguishable from one
// written before the columns existed - which is exactly what it means.
func TestLineageColumnsWithoutLineage(t *testing.T) {
	assert.False(t, toPgText(lineageBaseID(nil)).Valid)
	assert.False(t, toPgText(lineageRootID(nil)).Valid)
	assert.False(t, lineageAttempt(nil).Valid)
}
