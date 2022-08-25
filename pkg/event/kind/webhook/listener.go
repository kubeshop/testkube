package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	thttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var _ common.Listener = &WebhookListener{}

func NewWebhookListener(uri string, selector string, events []testkube.TestkubeEventType) *WebhookListener {
	return &WebhookListener{
		Uri:        uri,
		Log:        log.DefaultLogger,
		HttpClient: thttp.NewClient(),
		selector:   selector,
		events:     events,
	}
}

type WebhookListener struct {
	Uri        string
	Log        *zap.SugaredLogger
	HttpClient *http.Client
	events     []testkube.TestkubeEventType
	selector   string
}

func (l *WebhookListener) Selector() string {
	return l.selector
}

func (l *WebhookListener) Events() []testkube.TestkubeEventType {
	return l.events
}

func (l *WebhookListener) Notify(event testkube.TestkubeEvent) (result testkube.TestkubeEventResult) {
	body := bytes.NewBuffer([]byte{})
	err := json.NewEncoder(body).Encode(event)

	log := l.Log.With(event.Log()...)

	if err != nil {
		err = errors.Wrap(err, "webhook send json encode error")
		log.Errorw("webhook send json encode error", "error", err)
		return testkube.NewFailedTestkubeEventResult(event.Id, err)
	}

	request, err := http.NewRequest(http.MethodPost, l.Uri, body)
	if err != nil {
		log.Errorw("webhook request creating error", "error", err)
		return testkube.NewFailedTestkubeEventResult(event.Id, err)
	}

	resp, err := l.HttpClient.Do(request)
	if err != nil {
		log.Errorw("webhook send error", "error", err)
		return testkube.NewFailedTestkubeEventResult(event.Id, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorw("webhook read response error", "error", err)
		return testkube.NewFailedTestkubeEventResult(event.Id, err)
	}

	responseStr := string(data)

	if resp.StatusCode >= 400 {
		err := fmt.Errorf("webhook response with bad status code: %d", resp.StatusCode)
		log.Errorw("webhook send error", "error", err, "status", resp.StatusCode)
		return testkube.NewFailedTestkubeEventResult(event.Id, err).WithResult(responseStr)
	}

	log.Debugw("got webhook send result", "response", responseStr)
	return testkube.NewSuccessTestkubeEventResult(event.Id, responseStr)
}

func (l *WebhookListener) Kind() string {
	return "webhook"
}
