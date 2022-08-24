package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"

	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/slacknotifier"
)

const eventsBuffer = 10000
const workersCount = 20

// NewEmitter returns new emitter instance
func NewEmitter(webhooksClient *executorsclientv1.WebhooksClient) *Emitter {
	return &Emitter{
		Events:         make(chan testkube.TestkubeEvent, eventsBuffer),
		Responses:      make(chan WebhookResult, eventsBuffer),
		Log:            log.DefaultLogger,
		WebhooksClient: webhooksClient,
	}
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	WebhooksClient *executorsclientv1.WebhooksClient
	Events         chan testkube.TestkubeEvent
	Responses      chan WebhookResult
	Log            *zap.SugaredLogger
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
func (s *Emitter) Notify(event testkube.TestkubeEvent) {
	s.Log.Infow("notifying webhook", "eventURI", event.Uri, "eventType", event.Type_)
	s.Events <- event
}

// RunWorkers runs emitter workers responsible for sending HTTP requests
func (s *Emitter) RunWorkers() {
	s.Log.Debugw("Starting event emitter workers", "count", workersCount)
	for i := 0; i < workersCount; i++ {
		go s.Listen(s.Events)
	}
}

// Listen listens for webhook events
func (s *Emitter) Listen(events chan testkube.TestkubeEvent) {
	for event := range events {
		s.Log.Infow("processing event", event.Log()...)
		s.sendHttpEvent(event)
	}
}

// sendHttpEvent sends new webhook event - should be used when some event occurs
func (s *Emitter) sendHttpEvent(event testkube.TestkubeEvent) {
	body := bytes.NewBuffer([]byte{})
	err := json.NewEncoder(body).Encode(event)

	l := s.Log.With(event.Log()...)

	if err != nil {
		l.Errorw("webhook send json encode error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}

	request, err := http.NewRequest(http.MethodPost, event.Uri, body)
	if err != nil {
		l.Errorw("webhook request creating error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}

	// TODO use custom client with sane timeout values this one can starve queue in case of very slow clients
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		l.Errorw("webhook send error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}

	d, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Errorw("webhook read response error", "error", err)
		s.Responses <- WebhookResult{Error: err, Event: event}
		return
	}
	respBody := string(d)
	status := resp.StatusCode

	webhookResponse := WebhookHttpResponse{Body: respBody, StatusCode: status}
	l.Debugw("got webhook send result", "response", webhookResponse)
	s.Responses <- WebhookResult{Response: webhookResponse, Event: event}
}

func (s Emitter) NotifyAll(eventType *testkube.TestkubeEventType, execution testkube.Execution) error {
	webhookList, err := s.WebhooksClient.GetByEvent(eventType.String())
	if err != nil {
		return err
	}

	for _, wh := range webhookList.Items {
		sendEvent := wh.Spec.Selector == ""
		if !sendEvent {
			selector, err := labels.Parse(wh.Spec.Selector)
			if err != nil {
				return err
			}

			sendEvent = selector.Matches(labels.Set(execution.Labels))
		}

		if !sendEvent {
			continue
		}

		s.Log.Debugw("NotifyAll: Sending event", "uri", wh.Spec.Uri, "type", eventType, "executionID", execution.Id, "executionName", execution.Name)
		s.Notify(testkube.TestkubeEvent{
			Uri:       wh.Spec.Uri,
			Type_:     eventType,
			Execution: &execution,
		})
	}

	// TODO webhooks should be designed as events with type webhook/slack
	// TODO move it to Listen when the type webhook/slack is ready
	s.sendSlackEvent(eventType, execution)

	return nil

}

// TODO move it to EventEmitter as kind of SlackEvent
func (s Emitter) sendSlackEvent(eventType *testkube.TestkubeEventType, execution testkube.Execution) {
	err := slacknotifier.SendEvent(eventType, execution)
	if err != nil {
		s.Log.Warnw("notify slack failed", "error", err)
	}
}
