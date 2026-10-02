package scheduling

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

// deriveLineage decides where a rerun sits in its chain. The invariant that
// matters is that the root is carried down rather than reset: a rerun of a
// rerun still points at the original, so "every execution of chain R" stays a
// single query however deep the chain gets.
func TestDeriveLineage(t *testing.T) {
	for _, tc := range []struct {
		name string
		base testkube.TestWorkflowExecution
		want testkube.TestWorkflowExecutionLineage
	}{
		{
			name: "a rerun of an original run",
			base: testkube.TestWorkflowExecution{
				Id:      "exec-1",
				Lineage: &testkube.TestWorkflowExecutionLineage{RootId: "exec-1", Attempt: 1},
			},
			want: testkube.TestWorkflowExecutionLineage{BaseId: "exec-1", RootId: "exec-1", Attempt: 2},
		},
		{
			name: "a rerun of a rerun keeps the original as its root",
			base: testkube.TestWorkflowExecution{
				Id:      "exec-3",
				Lineage: &testkube.TestWorkflowExecutionLineage{BaseId: "exec-2", RootId: "exec-1", Attempt: 3},
			},
			want: testkube.TestWorkflowExecutionLineage{BaseId: "exec-3", RootId: "exec-1", Attempt: 4},
		},
		{
			// The upgrade path: rows written before lineage existed carry none,
			// and mean exactly what an original run means.
			name: "a base recorded before lineage existed",
			base: testkube.TestWorkflowExecution{Id: "exec-old"},
			want: testkube.TestWorkflowExecutionLineage{BaseId: "exec-old", RootId: "exec-old", Attempt: 2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, &tc.want, deriveLineage(&tc.base))
		})
	}
}

// EffectiveLineage is the single place the pre-lineage default lives, so both
// repositories agree on what an unrecorded lineage means.
func TestEffectiveLineage(t *testing.T) {
	t.Run("no lineage recorded means an original run", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{Id: "exec-1"}
		assert.Equal(t,
			testkube.TestWorkflowExecutionLineage{RootId: "exec-1", Attempt: 1},
			execution.EffectiveLineage())
	})

	t.Run("a recorded lineage is returned as it stands", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{
			Id:      "exec-2",
			Lineage: &testkube.TestWorkflowExecutionLineage{BaseId: "exec-1", RootId: "exec-1", Attempt: 2},
		}
		assert.Equal(t,
			testkube.TestWorkflowExecutionLineage{BaseId: "exec-1", RootId: "exec-1", Attempt: 2},
			execution.EffectiveLineage())
	})

	t.Run("a partially filled record still comes back coherent", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{
			Id:      "exec-3",
			Lineage: &testkube.TestWorkflowExecutionLineage{},
		}
		got := execution.EffectiveLineage()
		require.Equal(t, "exec-3", got.RootId)
		assert.Equal(t, int32(1), got.Attempt)
	})
}

// base_execution_id is defined as an id, but Repository.Get resolves by id or
// name. A name that happens to match an execution would otherwise derive the
// chain from whichever one answered to it, so anything but an exact id match is
// refused rather than quietly producing lineage for the wrong execution.
func TestPrepareExecutionsRejectsABaseThatIsNotAnId(t *testing.T) {
	ctrl := gomock.NewController(t)
	repository := testworkflow.NewMockRepository(ctrl)
	// The repository answers a name lookup with the execution it named.
	repository.EXPECT().Get(gomock.Any(), "my-workflow-7").
		Return(testkube.TestWorkflowExecution{Id: "exec-1", Name: "my-workflow-7"}, nil)

	enqueuer := &Enqueuer{executionRepository: repository}
	_, err := enqueuer.prepareExecutions(context.Background(), &cloud.ScheduleRequest{
		BaseExecutionId: common.Ptr("my-workflow-7"),
		Executions:      []*cloud.ScheduleExecution{{Selector: &cloud.ScheduleResourceSelector{Name: "wf"}}},
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an execution id")
}

// An exact id match is accepted and derives the chain as usual.
func TestPrepareExecutionsAcceptsAnExactBaseId(t *testing.T) {
	ctrl := gomock.NewController(t)
	repository := testworkflow.NewMockRepository(ctrl)
	repository.EXPECT().Get(gomock.Any(), "exec-1").
		Return(testkube.TestWorkflowExecution{Id: "exec-1", Name: "my-workflow-7"}, nil)

	enqueuer := &Enqueuer{executionRepository: repository}
	_, err := enqueuer.prepareExecutions(context.Background(), &cloud.ScheduleRequest{
		BaseExecutionId: common.Ptr("exec-1"),
		Executions:      nil,
	}, nil)

	require.NoError(t, err)
}
