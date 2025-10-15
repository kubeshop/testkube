package testworkflowexecutor

import (
	"context"
	"encoding/json"
	"io"
	"math"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/runner"
)

const (
	ConfigSizeLimit = 3 * 1024 * 1024
)

type TestWorkflowExecutionStream Stream[*testkube.TestWorkflowExecution]

//go:generate mockgen -destination=./mock_executor.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor" TestWorkflowExecutor
type TestWorkflowExecutor interface {
	Execute(ctx context.Context, environmentId string, req *cloud.ScheduleRequest) TestWorkflowExecutionStream
}

type executor struct {
	grpcClient           cloud.TestKubeCloudAPIClient
	apiKey               string
	organizationId       string
	defaultEnvironmentId string
	agentId              string
	emitter              event.Interface
	runner               runner.RunnerExecute
}

func New(
	grpcClient cloud.TestKubeCloudAPIClient,
	apiKey string,
	emitter event.Interface,
	runner runner.RunnerExecute,
	organizationId string,
	defaultEnvironmentId string,
	agentId string,
) TestWorkflowExecutor {
	return &executor{
		agentId:              agentId,
		grpcClient:           grpcClient,
		apiKey:               apiKey,
		emitter:              emitter,
		runner:               runner,
		organizationId:       organizationId,
		defaultEnvironmentId: defaultEnvironmentId,
	}
}

func (e *executor) Execute(ctx context.Context, environmentId string, req *cloud.ScheduleRequest) TestWorkflowExecutionStream {
	if environmentId == "" {
		environmentId = e.defaultEnvironmentId
	}

	ch := make(chan *testkube.TestWorkflowExecution)
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	ctx = agentclient.AddAPIKeyMeta(ctx, e.apiKey)
	ctx = metadata.AppendToOutgoingContext(ctx, "environment-id", environmentId)
	ctx = metadata.AppendToOutgoingContext(ctx, "organization-id", e.organizationId)
	ctx = metadata.AppendToOutgoingContext(ctx, "agent-id", e.agentId)
	resp, err := e.grpcClient.ScheduleExecution(ctx, req, opts...)
	resultStream := NewStream(ch)
	if err != nil {
		close(ch)
		resultStream.addError(err)
		return resultStream
	}
	go func() {
		defer close(ch)
		var item *cloud.ScheduleResponse
		for {
			item, err = resp.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					resultStream.addError(err)
				}
				break
			}
			var r testkube.TestWorkflowExecution
			err = json.Unmarshal(item.Execution, &r)
			if err != nil {
				resultStream.addError(err)
				break
			}
			ch <- &r
		}
	}()
	return resultStream
}
