package event

import (
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

const eventsBuffer = 10000
const workersCount = 20

// NewEmitter returns new emitter instance
func NewEmitter() *Emitter {
	return &Emitter{
		Events:  make(chan testkube.TestkubeEvent, eventsBuffer),
		Results: make(chan testkube.TestkubeEventResult, eventsBuffer),
		Log:     log.DefaultLogger,
	}
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	Events    chan testkube.TestkubeEvent
	Results   chan testkube.TestkubeEventResult
	Listeners []Listener
	Log       *zap.SugaredLogger
}

// WebhookResult is a wrapper for results from HTTP client for given webhook
type WebhookResult struct {
	Event    testkube.TestkubeEvent
	Error    error
	Response WebhookHttpResponse
}

// WebhookHttpResponse hold body and result of webhook response
type WebhookHttpResponse struct {
	StatusCode int
	Body       string
}

// Notify notifies emitter with webhook
func (s *Emitter) Register(listener Listener) {
	s.Listeners = append(s.Listeners, listener)
}

// Notify notifies emitter with webhook
func (s *Emitter) Notify(event testkube.TestkubeEvent) {
	s.Events <- event
}

// RunWorkers runs emitter workers responsible for sending HTTP requests
func (s *Emitter) RunWorkers() {
	s.Log.Debugw("Starting event emitter workers", "count", workersCount)
	for i := 0; i < workersCount; i++ {
		go s.RunWorker(s.Events, s.Results)
	}
}

func (s *Emitter) RunWorker(events chan testkube.TestkubeEvent, result chan testkube.TestkubeEventResult) {
	// TODO consider scaling this part to goroutines - for now we can just scale workers
	for event := range events {
		s.Log.Infow("processing event", event.Log()...)
		for _, listener := range s.Listeners {
			result <- listener.Notify(event)
		}
	}
}
