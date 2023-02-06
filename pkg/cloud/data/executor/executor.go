package executor

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
)

type Command string

//go:generate mockgen -destination=./mock_executor.go -package=executor "github.com/kubeshop/testkube/pkg/cloud/data/executor" Executor
type Executor interface {
	Execute(ctx context.Context, command Command, payload any) (response []byte, err error)
}

type CloudGRPCExecutor struct {
	client cloud.TestKubeCloudAPIClient
	apiKey string
}

func NewCloudGRPCExecutor(client cloud.TestKubeCloudAPIClient, apiKey string) *CloudGRPCExecutor {
	return &CloudGRPCExecutor{client: client, apiKey: apiKey}
}

func (e *CloudGRPCExecutor) Execute(ctx context.Context, command Command, payload any) (response []byte, err error) {
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
	ctx = agent.AddAPIKeyMeta(ctx, e.apiKey)
	var opts []grpc.CallOption
	cmdResponse, err := e.client.Call(ctx, &req, opts...)
	if err != nil {
		return nil, err
	}
	return cmdResponse.Response, nil
}
