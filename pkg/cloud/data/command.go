package data

import (
	"context"
	"encoding/json"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type Command string

const (
	CmdResultGetNextExecutionNumber Command = "result_get_next_execution_number"
	CmdResultGet                    Command = "result_get"
	CmdResultGetByNameAndTest       Command = "result_get_by_name_and_test"
	CmdResultGetLatestByTest        Command = "result_get_latest_by_test"
	CmdResultGetLatestByTests       Command = "result_get_latest_by_tests"
	CmdResultGetExecutions          Command = "result_get_executions"
	CmdResultGetExecutionTotals     Command = "result_get_execution_totals"
	CmdResultInsert                 Command = "result_insert"
	CmdResultUpdate                 Command = "result_update"
	CmdResultUpdateResult           Command = "result_update_result"
	CmdResultStartExecution         Command = "result_start_execution"
	CmdResultEndExecution           Command = "result_end_execution"
	CmdResultGetLabels              Command = "result_get_labels"
	CmdResultDeleteByTest           Command = "result_delete_by_test"
	CmdResultDeleteByTestSuite      Command = "result_delete_by_test_suite"
	CmdResultDeleteAll              Command = "result_delete_all"
	CmdResultDeleteByTests          Command = "result_delete_by_tests"
	CmdResultDeleteByTestSuites     Command = "result_delete_by_test_suites"
	CmdResultDeleteForAllTestSuites Command = "result_delete_for_all_test_suites"
	CmdResultGetTestMetrics         Command = "result_get_test_metrics"
)

type CommandRequest struct {
	Command Command `json:"command"`
	Payload any     `json:"payload"`
}

func execute(ctx context.Context, client cloud.TestKubeCloudAPIClient, command Command, payload any, apiKey string) (*cloud.CommandResponse, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return nil, err
	}
	req := cloud.CommandRequest{
		Command: string(command),
		Payload: &s,
	}
	ctx = agent.AddAPIKeyMeta(ctx, apiKey)
	var opts []grpc.CallOption
	response, err := client.Call(ctx, &req, opts...)
	if err != nil {
		return nil, err
	}
	return response, nil
}
