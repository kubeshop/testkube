// TODO how to handle pod issues here?
// TODO how to know that there is topic which is not handled by any subscriber?
// TODO we would need to check pending log topics and handle them after restart in case of log pod disaster

package logs

import (
	"context"
	"net/http"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/consumer"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

const (
	DefaultHttpAddress = ":8080"
)

func NewLogsService(nats *nats.Conn, js jetstream.JetStream, state state.Interface) *LogsService {
	return &LogsService{
		nats:              nats,
		adapters:          []consumer.Adapter{},
		js:                js,
		log:               log.DefaultLogger.With("service", "logs-service"),
		Ready:             make(chan struct{}, 1),
		httpAddress:       DefaultHttpAddress,
		consumerInstances: sync.Map{},
		state:             state,
	}
}

type LogsService struct {
	log      *zap.SugaredLogger
	nats     *nats.Conn
	js       jetstream.JetStream
	adapters []consumer.Adapter

	Ready chan struct{}

	// httpAddress is address for Kubernetes http health check handler
	httpAddress string
	httpServer  *http.Server

	// consumerInstances is internal executionID => Consumer map which we need to clean
	// each pod can have different executionId set of consumers
	consumerInstances sync.Map

	// state manager for keeping logs state (pending, finished)
	// will allow to distiguish from where load data from in OSS
	// cloud will be loading always them locally
	state state.Interface
}

// AddAdapter adds new adapter to logs service adapters will be configred based on given mode
// e.g. cloud mode will get cloud adapter to store logs directly on the cloud
func (ls *LogsService) AddAdapter(a consumer.Adapter) {
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
