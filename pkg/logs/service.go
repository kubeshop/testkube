// TODO how to handle pod issues here?
// TODO how to know that there is topic which is not handled by any subscriber?
// TODO we would need to check pending log topics and handle them after restart in case of log pod disaster

package logs

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/adapter"
	"github.com/kubeshop/testkube/pkg/logs/pb"
	"github.com/kubeshop/testkube/pkg/logs/repository"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

const (
	DefaultHttpAddress = ":8080"
	DefaultGrpcAddress = ":9090"

	DefaultStopWaitTime = 60 * time.Second // when stop event is faster than first message arrived
)

func NewLogsService(nats *nats.Conn, js jetstream.JetStream, state state.Interface) *LogsService {
	return &LogsService{
		nats:              nats,
		adapters:          []adapter.Adapter{},
		js:                js,
		log:               log.DefaultLogger.With("service", "logs-service"),
		Ready:             make(chan struct{}, 1),
		httpAddress:       DefaultHttpAddress,
		grpcAddress:       DefaultGrpcAddress,
		consumerInstances: sync.Map{},
		state:             state,
		stopWaitTime:      DefaultStopWaitTime,
	}
}

type LogsService struct {
	logsRepositoryFactory repository.Factory
	log                   *zap.SugaredLogger
	nats                  *nats.Conn
	js                    jetstream.JetStream
	adapters              []adapter.Adapter

	Ready chan struct{}

	// grpcAddress is address for grpc server
	grpcAddress string
	// grpcServer is grpc server for logs service
	grpcServer *grpc.Server

	// httpAddress is address for Kubernetes http health check handler
	httpAddress string
	// httpServer is http server for health check (for Kubernetes below 1.25)
	httpServer *http.Server

	// consumerInstances is internal executionID => Consumer map which we need to clean
	// each pod can have different executionId set of consumers
	consumerInstances sync.Map

	// state manager for keeping logs state (pending, finished)
	// will allow to distiguish from where load data from in OSS
	// cloud will be loading always them locally
	state state.Interface

	// stop wait time for messages cool down
	stopWaitTime time.Duration
}

// AddAdapter adds new adapter to logs service adapters will be configred based on given mode
// e.g. cloud mode will get cloud adapter to store logs directly on the cloud
func (ls *LogsService) AddAdapter(a adapter.Adapter) {
	ls.adapters = append(ls.adapters, a)
}

func (ls *LogsService) Run(ctx context.Context) (err error) {
	ls.log.Infow("starting logs service")

	// Handle start and stop events from nats
	// assuming after start event something is pushing data to the stream
	// it can be our handler or some other service

	// For start event we must build stream for given execution id and start consuming it
	// this one will must follow a queue group each pod will get it's own bunch of executions to handle
	// Start event will be triggered by logs process controller (scheduler)
	ls.nats.QueueSubscribe(StartSubject, StartQueue, ls.handleStart(ctx))

	// listen on all pods as we don't control which one will have given consumer
	// Stop event will be triggered by logs process controller (scheduler)
	ls.nats.Subscribe(StopSubject, ls.handleStop(ctx))

	// Send ready signal
	ls.Ready <- struct{}{}

	// block main routine
	<-ctx.Done()

	return nil
}

// TODO handle TLS
func (ls *LogsService) RunGRPCServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", ls.grpcAddress)
	if err != nil {
		return err
	}

	ls.grpcServer = grpc.NewServer()
	pb.RegisterLogsServiceServer(ls.grpcServer, NewLogsServer(ls.logsRepositoryFactory, ls.state))

	ls.log.Infow("starting grpc server", "address", ls.grpcAddress)
	return ls.grpcServer.Serve(lis)
}

func (ls *LogsService) Shutdown(ctx context.Context) (err error) {
	err = ls.httpServer.Shutdown(ctx)
	if err != nil {
		return err
	}

	if ls.grpcServer != nil {
		ls.grpcServer.GracefulStop()
	}

	// TODO decide how to handle graceful shutdown of consumers

	return nil
}

func (ls *LogsService) WithHttpAddress(address string) *LogsService {
	ls.httpAddress = address
	return ls
}

func (ls *LogsService) WithGrpcAddress(address string) *LogsService {
	ls.grpcAddress = address
	return ls
}

func (ls *LogsService) WithStopWaitTime(duration time.Duration) *LogsService {
	ls.stopWaitTime = duration
	return ls
}

func (ls *LogsService) WithRandomPort() *LogsService {
	port := rand.Intn(1000) + 17000
	ls.httpAddress = fmt.Sprintf("127.0.0.1:%d", port)
	port = rand.Intn(1000) + 18000
	ls.grpcAddress = fmt.Sprintf("127.0.0.1:%d", port)
	return ls
}

func (ls *LogsService) WithLogsRepositoryFactory(f repository.Factory) *LogsService {
	ls.logsRepositoryFactory = f
	return ls
}
