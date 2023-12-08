package logs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/repository"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

func TestGRPC_Server(t *testing.T) {
	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	state := &StateMock{state: state.LogStatePending}

	ls := NewLogsService(nil, nil, state).
		WithLogsRepositoryFactory(LogsFactoryMock{}).
		WithRandomPort()

	go ls.RunGRPCServer(ctx)

	count := 0

	stream := client.NewGrpcClient(ls.grpcAddress)
	ch := stream.Get(ctx, "id1")

	t.Log("waiting for logs")

	for l := range ch {
		t.Log(l)
		count++
	}

	assert.Equal(t, 10, count)
}

type StateMock struct {
	state state.LogState
}

func (s StateMock) Get(ctx context.Context, key string) (state.LogState, error) {
	return s.state, nil
}
func (s *StateMock) Put(ctx context.Context, key string, state state.LogState) error {
	s.state = state
	return nil
}

type LogsFactoryMock struct {
}

func (l LogsFactoryMock) GetRepository(state state.LogState) (repository.LogsRepository, error) {
	return LogsRepositoryMock{}, nil
}

type LogsRepositoryMock struct{}

func (l LogsRepositoryMock) Get(ctx context.Context, id string) chan events.LogResponse {
	ch := make(chan events.LogResponse, 10)
	defer close(ch)

	for i := 0; i < 100000; i++ {
		ch <- events.LogResponse{Log: events.Log{Time: time.Now(), Content: fmt.Sprintf("test %d", i), Error: false, Type: "test", Source: "test", Metadata: map[string]string{"test": "test"}}}
	}
	return ch
}
