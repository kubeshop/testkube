package runner

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/store"
)

const (
	ExecutionSaverUpdateRetryCount = 10
	ExecutionSaverUpdateRetryDelay = 300 * time.Millisecond
)

//go:generate mockgen -destination=./mock_executionsaver.go -package=runner "github.com/kubeshop/testkube/pkg/runner" ExecutionSaver
type ExecutionSaver interface {
	UpdateResult(result testkube.TestWorkflowResult)
	AppendOutput(output ...testkube.TestWorkflowOutput)
	End(ctx context.Context, result testkube.TestWorkflowResult) error
}

type executionSaver struct {
	id                   string
	organizationId       string
	environmentId        string
	runnerId             string
	executionsRepository testworkflow.Repository
	client               cloud.TestKubeCloudAPIClient
	grpcApiToken         string
	logs                 ExecutionLogsWriter
	newExecutionsEnabled bool

	// Intermediate data
	output       []testkube.TestWorkflowOutput
	result       *testkube.TestWorkflowResult
	resultUpdate store.Update
	resultMu     sync.Mutex

	outputSaved *atomic.Bool

	ctx       context.Context
	ctxCancel context.CancelFunc
}

func NewExecutionSaver(
	ctx context.Context,
	executionsRepository testworkflow.Repository,
	grpcClient cloud.TestKubeCloudAPIClient,
	grpcApiToken string,
	id string,
	organizationId string,
	environmentId string,
	runnerId string,
	logs ExecutionLogsWriter,
	newExecutionsEnabled bool,
) (ExecutionSaver, error) {
	ctx, cancel := context.WithCancel(ctx)
	outputSaved := atomic.Bool{}
	outputSaved.Store(true)
	saver := &executionSaver{
		id:                   id,
		organizationId:       organizationId,
		environmentId:        environmentId,
		runnerId:             runnerId,
		executionsRepository: executionsRepository,
		client:               grpcClient,
		grpcApiToken:         grpcApiToken,
		logs:                 logs,
		newExecutionsEnabled: newExecutionsEnabled,
		resultUpdate:         store.NewUpdate(),
		outputSaved:          &outputSaved,
		ctx:                  ctx,
		ctxCancel:            cancel,
	}
	go saver.watchResultUpdates()

	return saver, nil
}

func (s *executionSaver) watchResultUpdates() {
	defer s.resultUpdate.Close()
	ch := s.resultUpdate.Channel(s.ctx)
	var prev *testkube.TestWorkflowResult
	for {
		select {
		case <-s.ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			for i := 0; i < ExecutionSaverUpdateRetryCount; i++ {
				s.resultMu.Lock()
				next := s.result
				s.resultMu.Unlock()
				if prev == next {
					break
				}
				err := s.saveResult(s.ctx, next)
				if err == nil {
					break
				}
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(ExecutionSaverUpdateRetryDelay):
				}
			}
		}
	}
}

func (s *executionSaver) UpdateResult(result testkube.TestWorkflowResult) {
	s.resultMu.Lock()
	defer s.resultMu.Unlock()
	s.result = &result
	s.resultUpdate.Emit()
}

func (s *executionSaver) AppendOutput(output ...testkube.TestWorkflowOutput) {
	s.output = append(s.output, output...)
	s.outputSaved.Store(false)
}

func (s *executionSaver) saveOutput(ctx context.Context) error {
	// TODO: Consider AppendOutput ($push) instead
	if !s.newExecutionsEnabled {
		return s.executionsRepository.UpdateOutput(ctx, s.id, s.output)
	}

	output := common.MapSlice(s.output, func(t testkube.TestWorkflowOutput) *cloud.ExecutionOutput {
		v, _ := json.Marshal(t)
		return &cloud.ExecutionOutput{
			Ref:   t.Ref,
			Name:  t.Name,
			Value: v,
		}
	})
	md := metadata.New(map[string]string{apiKeyMeta: s.grpcApiToken, orgIdMeta: s.organizationId, agentIdMeta: s.runnerId})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	_, err := s.client.UpdateExecutionOutput(metadata.NewOutgoingContext(ctx, md), &cloud.UpdateExecutionOutputRequest{
		EnvironmentId: s.environmentId,
		Id:            s.id,
		Output:        output,
	}, opts...)
	return err
}

func (s *executionSaver) saveResult(ctx context.Context, result *testkube.TestWorkflowResult) error {
	if !s.newExecutionsEnabled {
		return s.executionsRepository.UpdateResult(ctx, s.id, result)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	md := metadata.New(map[string]string{apiKeyMeta: s.grpcApiToken, orgIdMeta: s.organizationId, agentIdMeta: s.runnerId})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	_, err = s.client.UpdateExecutionResult(metadata.NewOutgoingContext(ctx, md), &cloud.UpdateExecutionResultRequest{
		EnvironmentId: s.environmentId,
		Id:            s.id,
		Result:        resultBytes,
	}, opts...)
	return err
}

func (s *executionSaver) End(ctx context.Context, result testkube.TestWorkflowResult) error {
	s.ctxCancel()
	s.resultMu.Lock()
	defer s.resultMu.Unlock()

	// Save the logs and output
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		if s.outputSaved.Load() {
			return nil
		}
		return s.saveOutput(ctx)
	})
	g.Go(func() error {
		if s.logs.Saved() {
			return nil
		}
		return s.logs.Save(ctx)
	})
	err := g.Wait()
	if err != nil {
		return err
	}

	// Save the final result
	if s.newExecutionsEnabled {
		err = s.saveFinalResult(ctx, &result)
	} else {
		err = s.saveResult(ctx, &result)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *executionSaver) saveFinalResult(ctx context.Context, result *testkube.TestWorkflowResult) error {
	if result == nil {
		return errors.New("missing result")
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	md := metadata.New(map[string]string{apiKeyMeta: s.grpcApiToken, orgIdMeta: s.organizationId, agentIdMeta: s.runnerId})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	_, err = s.client.FinishExecution(metadata.NewOutgoingContext(ctx, md), &cloud.FinishExecutionRequest{
		EnvironmentId: s.environmentId,
		Id:            s.id,
		Result:        resultBytes,
	}, opts...)
	return err
}
