package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const eventsBuffer = 1000
const workersCount = 20

func NewServer() *Server {
	return &Server{
		Events:    make(chan testkube.WebhookEvent, eventsBuffer),
		Responses: make(chan WebhookResult, eventsBuffer),
	}
}

type Server struct {
	Events    chan testkube.WebhookEvent
	Responses chan WebhookResult
}

type WebhookResult struct {
	Error    error
	Response *http.Response
}

func (s *Server) Notify(event testkube.WebhookEvent, uris []string) {
	s.Events <- event
}

func (s *Server) RunWorkers() {
	for i := 0; i < workersCount; i++ {
		go s.Listen(s.Events)
	}
}

func (s *Server) Listen(events chan testkube.WebhookEvent) {
	for event := range events {
		s.Send(event)
	}
}

func (s *Server) Send(event testkube.WebhookEvent) {
	b := bytes.NewBuffer([]byte{})
	err := json.NewEncoder(b).Encode(event)
	if err != nil {
		s.Responses <- WebhookResult{Error: err}
		return
	}

	request, err := http.NewRequest(http.MethodPost, event.Uri, b)
	if err != nil {
		s.Responses <- WebhookResult{Error: err}
		return
	}

	// TODO use custom client with sane timeout values this one can starve queue in case of very slow clients
	response, err := http.DefaultClient.Do(request)
	s.Responses <- WebhookResult{Error: err, Response: response}
}
