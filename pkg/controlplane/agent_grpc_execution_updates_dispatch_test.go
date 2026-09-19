package controlplane

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
)

// fakeExecutionQuerier records what the handler asked for and returns what the
// test staged.
type fakeExecutionQuerier struct {
	transitions    []scheduling.ExecutionTransition
	transitionsErr error

	toStart    []testkube.TestWorkflowExecution
	toStartErr error

	stale []string

	gotLimit            int
	gotRedispatchBefore time.Time
	toStartCalls        int
}

func (f *fakeExecutionQuerier) Transitions(context.Context) ([]scheduling.ExecutionTransition, error) {
	return f.transitions, f.transitionsErr
}

func (f *fakeExecutionQuerier) ToStart(_ context.Context, limit int, redispatchBefore time.Time) ([]testkube.TestWorkflowExecution, error) {
	f.toStartCalls++
	f.gotLimit = limit
	f.gotRedispatchBefore = redispatchBefore
	if f.toStartErr != nil {
		return nil, f.toStartErr
	}
	if len(f.toStart) > limit {
		return f.toStart[:limit], nil
	}
	return f.toStart, nil
}

func (f *fakeExecutionQuerier) StaleStarting(context.Context, time.Time, int) ([]string, error) {
	return f.stale, nil
}

// fakeController records the dispatch bookkeeping.
type fakeController struct {
	started   []string
	refreshed []string
	calls     int
}

func (f *fakeController) StartExecutions(_ context.Context, ids []string) error {
	f.calls++
	f.started = append(f.started, ids...)
	return nil
}

func (f *fakeController) RefreshStartingExecutions(_ context.Context, ids []string) error {
	f.calls++
	f.refreshed = append(f.refreshed, ids...)
	return nil
}

func (f *fakeController) PauseExecution(context.Context, string) error       { return nil }
func (f *fakeController) ResumeExecution(context.Context, string) error      { return nil }
func (f *fakeController) AbortExecution(context.Context, string) error       { return nil }
func (f *fakeController) CancelExecution(context.Context, string) error      { return nil }
func (f *fakeController) ForceCancelExecution(context.Context, string) error { return nil }

func newDispatchServer(t *testing.T, querier *fakeExecutionQuerier, controller *fakeController, cfg Config) *Server {
	t.Helper()
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop().Sugar()
	}
	return &Server{
		cfg:                 cfg,
		executionQuerier:    querier,
		ExecutionController: controller,
	}
}

func assignedExecution(id string) testkube.TestWorkflowExecution {
	return testkube.TestWorkflowExecution{
		Id:       id,
		Result:   &testkube.TestWorkflowResult{Status: common.Ptr(testkube.ASSIGNED_TestWorkflowStatus)},
		Workflow: &testkube.TestWorkflow{Name: "wf-" + id},
	}
}

func startingExecution(id string) testkube.TestWorkflowExecution {
	return testkube.TestWorkflowExecution{
		Id:       id,
		Result:   &testkube.TestWorkflowResult{Status: common.Ptr(testkube.STARTING_TestWorkflowStatus)},
		Workflow: &testkube.TestWorkflow{Name: "wf-" + id},
	}
}

// A user cancel must reach the runner as a cancel.
//
// CancelExecutionRunningResult writes predicted_status 'canceled' and
// AbortExecutionRunningResult writes 'aborted'; the runner maps CANCELLED to
// Cancel and ABORTED to Abort. The handler had these two exchanged, so a cancel
// ran an abort and an abort ran a cancel. This is the counterpart to
// pkg/runner/grpc/client_stop_test.go, which pins the runner half.
func TestGetExecutionUpdates_MapsStoppingToThePredictedOutcome(t *testing.T) {
	for _, tc := range []struct {
		name      string
		predicted testkube.TestWorkflowStatus
		want      executionv1.ExecutionState
	}{
		{"a cancel cancels", testkube.CANCELED_TestWorkflowStatus, executionv1.ExecutionState_EXECUTION_STATE_CANCELLED},
		{"an abort aborts", testkube.ABORTED_TestWorkflowStatus, executionv1.ExecutionState_EXECUTION_STATE_ABORTED},
		{"an unset prediction aborts", "", executionv1.ExecutionState_EXECUTION_STATE_ABORTED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			querier := &fakeExecutionQuerier{transitions: []scheduling.ExecutionTransition{{
				Id:              "exec-1",
				Status:          testkube.STOPPING_TestWorkflowStatus,
				PredictedStatus: tc.predicted,
			}}}
			server := newDispatchServer(t, querier, &fakeController{}, Config{})

			response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

			require.NoError(t, err)
			require.Len(t, response.GetUpdate(), 1)
			assert.Equal(t, tc.want, response.GetUpdate()[0].GetTransitionTo())
			assert.Equal(t, "exec-1", response.GetUpdate()[0].GetExecutionId())
		})
	}
}

func TestGetExecutionUpdates_MapsPauseAndResume(t *testing.T) {
	querier := &fakeExecutionQuerier{transitions: []scheduling.ExecutionTransition{
		{Id: "exec-1", Status: testkube.PAUSING_TestWorkflowStatus},
		{Id: "exec-2", Status: testkube.RESUMING_TestWorkflowStatus},
	}}
	server := newDispatchServer(t, querier, &fakeController{}, Config{})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	require.Len(t, response.GetUpdate(), 2)
	assert.Equal(t, executionv1.ExecutionState_EXECUTION_STATE_PAUSED, response.GetUpdate()[0].GetTransitionTo())
	assert.Equal(t, executionv1.ExecutionState_EXECUTION_STATE_RUNNING, response.GetUpdate()[1].GetTransitionTo())
}

// The dispatch batch is bounded. Unbounded, a burst of workflows sharing a cron
// minute made one poll exceed the runner's call deadline.
func TestGetExecutionUpdates_BoundsTheDispatchBatch(t *testing.T) {
	querier := &fakeExecutionQuerier{}
	for i := 0; i < 200; i++ {
		querier.toStart = append(querier.toStart, assignedExecution(fmt.Sprintf("exec-%d", i)))
	}
	server := newDispatchServer(t, querier, &fakeController{}, Config{DispatchBatchSize: 25})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	assert.Equal(t, 25, querier.gotLimit, "the handler must ask for a bounded page")
	assert.Len(t, response.GetStart(), 25)
}

func TestGetExecutionUpdates_AppliesDefaultsOnAZeroConfig(t *testing.T) {
	querier := &fakeExecutionQuerier{}
	// A zero Config, as a struct-literal Server has.
	server := newDispatchServer(t, querier, &fakeController{}, Config{})

	before := time.Now()
	_, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})
	require.NoError(t, err)

	assert.Equal(t, DefaultDispatchBatchSize, querier.gotLimit)
	assert.WithinDuration(t, before.Add(-DefaultRedispatchAfter), querier.gotRedispatchBefore, time.Minute,
		"STARTING rows must only be re-offered once their lease has expired")
}

// Newly assigned rows are claimed; rows already STARTING have their lease
// renewed instead, so a retried execution is not re-offered on the next poll.
func TestGetExecutionUpdates_ClaimsAssignedAndRenewsRedispatched(t *testing.T) {
	querier := &fakeExecutionQuerier{toStart: []testkube.TestWorkflowExecution{
		assignedExecution("new-1"),
		startingExecution("retry-1"),
		assignedExecution("new-2"),
		startingExecution("retry-2"),
	}}
	controller := &fakeController{}
	server := newDispatchServer(t, querier, controller, Config{})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	assert.Len(t, response.GetStart(), 4, "every row in the batch is offered to the runner")
	assert.Equal(t, []string{"new-1", "new-2"}, controller.started)
	assert.Equal(t, []string{"retry-1", "retry-2"}, controller.refreshed)
}

// The lean projection has to be enough to build a complete ExecutionStart, or
// the poll would have to hydrate and we are back to 1+6N queries.
func TestGetExecutionUpdates_BuildsStartFromTheLeanProjection(t *testing.T) {
	querier := &fakeExecutionQuerier{toStart: []testkube.TestWorkflowExecution{{
		Id:              "exec-1",
		GroupId:         "group-1",
		Name:            "wf-a-7",
		Number:          7,
		ScheduledAt:     time.Unix(1735689600, 0),
		DisableWebhooks: true,
		Tags:            map[string]string{"env": "prod"},
		Result:          &testkube.TestWorkflowResult{Status: common.Ptr(testkube.ASSIGNED_TestWorkflowStatus)},
		Workflow:        &testkube.TestWorkflow{Name: "wf-a"},
		Lineage:         &testkube.TestWorkflowExecutionLineage{BaseId: "exec-0", RootId: "exec-0", Attempt: 2},
		// Deliberately nil: the runner fetches the workflow itself.
		ResolvedWorkflow: nil,
		Signature:        nil,
	}}}
	server := newDispatchServer(t, querier, &fakeController{}, Config{})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	require.Len(t, response.GetStart(), 1)
	start := response.GetStart()[0]
	assert.Equal(t, "exec-1", start.GetExecutionId())
	assert.Equal(t, "group-1", start.GetGroupId())
	assert.Equal(t, "wf-a-7", start.GetName())
	assert.Equal(t, int32(7), start.GetNumber())
	assert.Equal(t, "wf-a", start.GetWorkflowName())
	assert.True(t, start.GetDisableWebhooks())
	assert.Equal(t, map[string]string{"env": "prod"}, start.GetTags())
	require.NotNil(t, start.GetLineage())
	assert.Equal(t, "exec-0", start.GetLineage().GetBaseExecutionId())
	assert.Equal(t, int32(2), start.GetLineage().GetAttempt())
}

// The RPC carries both control instructions and the dispatch batch, so a failure
// on one must not cost the other. This preserves the contract that the call
// never fails.
func TestGetExecutionUpdates_SurvivesAFailedTransitionsRead(t *testing.T) {
	querier := &fakeExecutionQuerier{
		transitionsErr: errors.New("boom"),
		toStart:        []testkube.TestWorkflowExecution{assignedExecution("exec-1")},
	}
	server := newDispatchServer(t, querier, &fakeController{}, Config{})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	assert.Empty(t, response.GetUpdate())
	assert.Len(t, response.GetStart(), 1, "a failed transitions read must not cost the dispatch batch")
}

func TestGetExecutionUpdates_SurvivesAFailedDispatchRead(t *testing.T) {
	querier := &fakeExecutionQuerier{
		transitions: []scheduling.ExecutionTransition{{Id: "exec-1", Status: testkube.PAUSING_TestWorkflowStatus}},
		toStartErr:  errors.New("boom"),
	}
	controller := &fakeController{}
	server := newDispatchServer(t, querier, controller, Config{})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	assert.Len(t, response.GetUpdate(), 1)
	assert.Empty(t, response.GetStart())
	assert.Empty(t, controller.started, "nothing may be claimed when the read failed")
}

// The cost of a poll must not grow with the backlog. This is the property the
// whole fix is about: it used to be one query plus six per row, plus two writes
// per assigned row, so a deep backlog made every poll slower than the last.
func TestGetExecutionUpdates_QueryCountIsIndependentOfBacklog(t *testing.T) {
	callsAt := func(backlog int) (reads int, writes int) {
		querier := &fakeExecutionQuerier{}
		for i := 0; i < backlog; i++ {
			querier.toStart = append(querier.toStart, assignedExecution(fmt.Sprintf("exec-%d", i)))
		}
		controller := &fakeController{}
		server := newDispatchServer(t, querier, controller, Config{DispatchBatchSize: 25})

		_, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})
		require.NoError(t, err)

		// One transitions read plus one dispatch read, regardless of depth.
		return 1 + querier.toStartCalls, controller.calls
	}

	shallowReads, shallowWrites := callsAt(10)
	deepReads, deepWrites := callsAt(1000)

	assert.Equal(t, shallowReads, deepReads, "a deeper backlog must not cost more reads")
	assert.Equal(t, shallowWrites, deepWrites, "a deeper backlog must not cost more writes")
}

// The response must not grow with the backlog either: its size is what had to
// fit inside the runner's 30s deadline.
func TestGetExecutionUpdates_ResponseSizeIsBounded(t *testing.T) {
	sizeAt := func(backlog int) int {
		querier := &fakeExecutionQuerier{}
		for i := 0; i < backlog; i++ {
			querier.toStart = append(querier.toStart, assignedExecution("exec-0"))
		}
		server := newDispatchServer(t, querier, &fakeController{}, Config{DispatchBatchSize: 25})

		response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})
		require.NoError(t, err)
		return proto.Size(response)
	}

	assert.Equal(t, sizeAt(30), sizeAt(1000),
		"the response is capped by the batch, not by the backlog")
}
