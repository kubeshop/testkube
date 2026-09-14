package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Storage keeps pre-lineage rows sparse and never backfills them, so the read
// path is what makes the contract true: every execution reports a lineage, an
// original run being its own root at attempt 1. Without this the API omits it
// for legacy rows while the pod synthesizes one, and a client cannot group a
// legacy original with the reruns that descend from it.
func TestApplyEffectiveLineageFillsInALegacyRow(t *testing.T) {
	execution := &TestWorkflowExecution{Id: "exec-1"}

	execution.ApplyEffectiveLineage()

	require.NotNil(t, execution.Lineage)
	assert.Empty(t, execution.Lineage.BaseId)
	assert.Equal(t, "exec-1", execution.Lineage.RootId)
	assert.Equal(t, int32(1), execution.Lineage.Attempt)
}

// A recorded lineage is left exactly as it stands.
func TestApplyEffectiveLineageLeavesARecordedChain(t *testing.T) {
	execution := &TestWorkflowExecution{
		Id:      "exec-3",
		Lineage: &TestWorkflowExecutionLineage{BaseId: "exec-2", RootId: "exec-1", Attempt: 3},
	}

	execution.ApplyEffectiveLineage()

	assert.Equal(t, "exec-2", execution.Lineage.BaseId)
	assert.Equal(t, "exec-1", execution.Lineage.RootId)
	assert.Equal(t, int32(3), execution.Lineage.Attempt)
}

// The summary has to answer the same way, or a list view disagrees with the
// detail view about the same execution.
func TestSummaryApplyEffectiveLineageMatchesTheExecution(t *testing.T) {
	summary := &TestWorkflowExecutionSummary{Id: "exec-1"}

	summary.ApplyEffectiveLineage()

	require.NotNil(t, summary.Lineage)
	assert.Empty(t, summary.Lineage.BaseId)
	assert.Equal(t, "exec-1", summary.Lineage.RootId)
	assert.Equal(t, int32(1), summary.Lineage.Attempt)
}

// A partially filled record comes back coherent field by field, the same way
// EffectiveLineage handles it.
func TestSummaryApplyEffectiveLineageFillsAPartialRecord(t *testing.T) {
	summary := &TestWorkflowExecutionSummary{
		Id:      "exec-2",
		Lineage: &TestWorkflowExecutionLineage{BaseId: "exec-1"},
	}

	summary.ApplyEffectiveLineage()

	assert.Equal(t, "exec-1", summary.Lineage.BaseId)
	assert.Equal(t, "exec-2", summary.Lineage.RootId)
	assert.Equal(t, int32(1), summary.Lineage.Attempt)
}
