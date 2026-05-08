package executor

import (
	"context"
	"encoding/json"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/types/known/structpb"

	intconfig "github.com/kubeshop/testkube/internal/config"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
)

type Command string

//go:generate go tool mockgen -destination=./mock_executor.go -package=executor "github.com/kubeshop/testkube/pkg/cloud/data/executor" Executor
type Executor interface {
	Execute(ctx context.Context, command Command, payload any) (response []byte, err error)
}

type CloudGRPCExecutor struct {
	client  cloud.TestKubeCloudAPIClient
	apiKey  string
	orgID   string
	envID   string
	agentID string
}

func NewCloudGRPCExecutor(client cloud.TestKubeCloudAPIClient, proContext *intconfig.ProContext) *CloudGRPCExecutor {
	return &CloudGRPCExecutor{client: client, apiKey: proContext.APIKey, orgID: proContext.OrgID, envID: proContext.EnvID, agentID: proContext.Agent.ID}
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
	ctx = agentclient.AddMetadata(ctx, e.apiKey, e.orgID, e.envID, e.agentID)
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	cmdResponse, err := e.client.Call(ctx, &req, opts...)
	if err != nil {
		return nil, err
	}
	return cmdResponse.Response, nil
}
