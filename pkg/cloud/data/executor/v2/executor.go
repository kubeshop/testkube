package executor

import (
	"context"
	"encoding/json"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
)

type ExecuteParams struct {
	Command Command
	Payload any
}

type Command string

var _ Executor[Command] = &CloudGRPCExecutor[Command]{}

//go:generate mockgen -destination=./mock_executor.go -package=executor "github.com/kubeshop/testkube/pkg/cloud/data/executor/v2" Executor
type Executor[T any] interface {
	Execute(ctx context.Context, params ExecuteParams) (response T, err error)
	Close() error
}

type CloudGRPCExecutor[T any] struct {
	client   cloud.TestKubeCloudAPIClient
	conn     *grpc.ClientConn
	apiKey   string
	runnerId string
}

func NewCloudGRPCExecutor[T any](client cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey, runnerId string) *CloudGRPCExecutor[T] {
	return &CloudGRPCExecutor[T]{client: client, conn: grpcConn, apiKey: apiKey, runnerId: runnerId}
}

func (e *CloudGRPCExecutor[T]) Execute(ctx context.Context, params ExecuteParams) (response T, err error) {
	jsonPayload, err := json.Marshal(params.Payload)
	if err != nil {
		return response, err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return response, err
	}
	req := cloud.CommandRequest{
		Command: string(params.Command),
		Payload: &s,
	}
	ctx = agent.AddContextMetadata(ctx, e.apiKey, e.runnerId)
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	cmdResponse, err := e.client.Call(ctx, &req, opts...)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(cmdResponse.Response, &response)

	return
}

func (e *CloudGRPCExecutor[T]) Close() error {
	return e.conn.Close()
}
