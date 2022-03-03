package webhook

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
)

const eventsBuffer = 10000
const workersCount = 20

func NewEmitter() *Emitter {
	return &Emitter{
		Events:    make(chan testkube.WebhookEvent, eventsBuffer),
		Responses: make(chan WebhookResult, eventsBuffer),
		Log:       log.DefaultLogger,
	}
}

type Emitter struct {
	Events    chan testkube.WebhookEvent
	Responses chan WebhookResult
	Log       *zap.SugaredLogger
}

type WebhookResult struct {
	Event    testkube.WebhookEvent
	Error    error
	Response WebhookHttpResponse
}

type WebhookHttpResponse struct {
	StatusCode int
	Body       string
}

func (s *Emitter) Notify(event testkube.WebhookEvent) {
	s.Log.Debugw("notifying webhook", "event", event)
	s.Events <- event
}

func (s *Emitter) RunWorkers() {
	for i := 0; i < workersCount; i++ {
		go s.Listen(s.Events)
	}
}

func (s *Emitter) Listen(events chan testkube.WebhookEvent) {
	for event := range events {
		s.Send(event)
	}
}

func (s *Emitter) Send(event testkube.WebhookEvent) {
	body := bytes.NewBuffer([]byte{})
	err := json.NewEncoder(body).Encode(event)

	if err != nil {
		s.Log.Errorw("webhook send json encode error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}

	request, err := http.NewRequest(http.MethodPost, event.Uri, body)
	if err != nil {
		s.Log.Errorw("webhook request creating error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}

	// TODO use custom client with sane timeout values this one can starve queue in case of very slow clients
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		s.Log.Errorw("webhook send error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.Log.Errorw("webhook read response error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}
	respBody := string(d)
	status := resp.StatusCode

	result := WebhookResult{Response: WebhookHttpResponse{Body: respBody, StatusCode: status}, Event: event}
	s.Log.Debugw("got webhook send result", "result", result)
	s.Responses <- result
}
