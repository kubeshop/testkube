package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func finishedExecution(statusValue testkube.TestWorkflowStatus) testkube.TestWorkflowExecution {
	return testkube.TestWorkflowExecution{
		Id:       "exec-1",
		RunnerId: common.StandaloneRunner,
		Result: &testkube.TestWorkflowResult{
			Status:     common.Ptr(statusValue),
			FinishedAt: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC),
		},
	}
}

// A late acceptance must not resurrect an execution that is already over.
//
// Init sets SCHEDULING unconditionally, so without the guard the reaper could
// abort a dispatch whose lease expired while the runner's Kubernetes setup was
// still in flight, and the runner's later acknowledgement would turn the
// terminal execution back into a running one - an aborted event followed by a
// live execution and a contradictory final result.
func TestAcceptExecution_RefusesAFinishedExecution(t *testing.T) {
	for _, finished := range []testkube.TestWorkflowStatus{
		testkube.ABORTED_TestWorkflowStatus,
		testkube.CANCELED_TestWorkflowStatus,
		testkube.FAILED_TestWorkflowStatus,
		testkube.PASSED_TestWorkflowStatus,
	} {
		t.Run(string(finished), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := testworkflow.NewMockRepository(ctrl)
			repo.EXPECT().Get(gomock.Any(), "exec-1").Return(finishedExecution(finished), nil)
			// Init must never be reached.
			repo.EXPECT().Init(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			server := &Server{resultsRepository: repo}

			_, err := server.AcceptExecution(context.Background(), &executionv1.AcceptExecutionRequest{
				ExecutionId: proto.String("exec-1"),
				Namespace:   proto.String("testkube"),
			})

			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, status.Code(err),
				"the runner has created resources by now and needs to be told to tear them down")
			assert.Contains(t, err.Error(), "already finished")
		})
	}
}

// The ordinary path is unaffected: an execution still in flight is accepted.
func TestAcceptExecution_AcceptsAnUnfinishedExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := testworkflow.NewMockRepository(ctrl)
	execution := testkube.TestWorkflowExecution{
		Id:       "exec-1",
		RunnerId: common.StandaloneRunner,
		Result:   &testkube.TestWorkflowResult{Status: common.Ptr(testkube.STARTING_TestWorkflowStatus)},
	}
	repo.EXPECT().Get(gomock.Any(), "exec-1").Return(execution, nil)
	repo.EXPECT().Init(gomock.Any(), "exec-1", gomock.Any()).Return(nil)

	server := &Server{resultsRepository: repo}

	_, err := server.AcceptExecution(context.Background(), &executionv1.AcceptExecutionRequest{
		ExecutionId: proto.String("exec-1"),
		Namespace:   proto.String("testkube"),
	})

	require.NoError(t, err)
}

// A status that has not finished yet - one with a terminal status but no
// FinishedAt, which IsFinished treats as still running - must not be refused.
func TestAcceptExecution_AcceptsWhenOnlyTheStatusLooksTerminal(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := testworkflow.NewMockRepository(ctrl)
	execution := testkube.TestWorkflowExecution{
		Id:       "exec-1",
		RunnerId: common.StandaloneRunner,
		// Terminal status, but never finished.
		Result: &testkube.TestWorkflowResult{Status: common.Ptr(testkube.ABORTED_TestWorkflowStatus)},
	}
	repo.EXPECT().Get(gomock.Any(), "exec-1").Return(execution, nil)
	repo.EXPECT().Init(gomock.Any(), "exec-1", gomock.Any()).Return(nil)

	server := &Server{resultsRepository: repo}

	_, err := server.AcceptExecution(context.Background(), &executionv1.AcceptExecutionRequest{
		ExecutionId: proto.String("exec-1"),
		Namespace:   proto.String("testkube"),
	})

	require.NoError(t, err)
}
