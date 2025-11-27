package testworkflowexecutor

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

type recordingEmitter struct {
	mu     sync.Mutex
	events []testkube.Event
}

func (e *recordingEmitter) Notify(event testkube.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, event)
}

func (e *recordingEmitter) Events() []testkube.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]testkube.Event, len(e.events))
	copy(cp, e.events)
	return cp
}

type scheduleExecutionServer struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	executions []*testkube.TestWorkflowExecution
}

func (s *scheduleExecutionServer) ScheduleExecution(_ *cloud.ScheduleRequest, srv cloud.TestKubeCloudAPI_ScheduleExecutionServer) error {
	for _, execution := range s.executions {
		payload, err := json.Marshal(execution)
		if err != nil {
			return err
		}
		if err = srv.Send(&cloud.ScheduleResponse{Execution: payload}); err != nil {
			return err
		}
	}
	return nil
}

func TestExecute_NewArchitectureEmitsQueueEvent(t *testing.T) {
	ctx := context.Background()

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	executionStatus := testkube.QUEUED_TestWorkflowStatus
	execution := &testkube.TestWorkflowExecution{
		Id:     "exec-1",
		Name:   "workflow-1",
		Result: &testkube.TestWorkflowResult{Status: &executionStatus},
	}
	cloud.RegisterTestKubeCloudAPIServer(server, &scheduleExecutionServer{
		executions: []*testkube.TestWorkflowExecution{execution},
	})
	go func() {
		_ = server.Serve(lis)
	}()
	t.Cleanup(server.Stop)

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	exec := &executor{
		apiKey:               "api-key",
		grpcClient:           cloud.NewTestKubeCloudAPIClient(conn),
		emitter:              &recordingEmitter{},
		organizationId:       "org-1",
		defaultEnvironmentId: "env-1",
		agentId:              "agent-1",
	}

	req := &cloud.ScheduleRequest{
		Executions: []*cloud.ScheduleExecution{
			{Selector: &cloud.ScheduleResourceSelector{Name: "workflow-1"}},
		},
	}

	executions, err := exec.Execute(ctx, req)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	require.Equal(t, execution.Id, executions[0].Id)
	require.Equal(t, execution.Name, executions[0].Name)
}
