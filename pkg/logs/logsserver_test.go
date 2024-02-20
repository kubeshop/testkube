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

const count = 10

func TestGRPC_Server(t *testing.T) {
	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	state := &StateMock{state: state.LogStatePending}

	ls := NewLogsService(nil, nil, state, nil).
		WithLogsRepositoryFactory(LogsFactoryMock{}).
		WithRandomPort()

	go ls.RunGRPCServer(ctx, nil)

	// allow server to splin up
	time.Sleep(time.Millisecond * 100)

	expectedCount := 0

	stream := client.NewGrpcClient(ls.grpcAddress, nil)
	ch, err := stream.Get(ctx, "id1")
	assert.NoError(t, err)

	t.Log("waiting for logs")

	for l := range ch {
		t.Log(l)
		expectedCount++
	}

	assert.Equal(t, count, expectedCount)
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

func (l LogsRepositoryMock) Get(ctx context.Context, id string) (chan events.LogResponse, error) {
	ch := make(chan events.LogResponse, 10)
	defer close(ch)

	for i := 0; i < count; i++ {
		ch <- events.LogResponse{Log: events.Log{Time: time.Now(), Content: fmt.Sprintf("test %d", i), Error_: false, Type_: "test", Source: "test", Metadata: map[string]string{"test": "test"}}}
	}
	return ch, nil
}
