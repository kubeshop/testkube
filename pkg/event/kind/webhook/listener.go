package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	thttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/log"
)

var _ common.Listener = (*WebhookListener)(nil)

func NewWebhookListener(name, uri, selector string, events []testkube.EventType,
	payloadObjectField, payloadTemplate string, headers map[string]string) *WebhookListener {
	return &WebhookListener{
		name:               name,
		Uri:                uri,
		Log:                log.DefaultLogger,
		HttpClient:         thttp.NewClient(),
		selector:           selector,
		events:             events,
		payloadObjectField: payloadObjectField,
		payloadTemplate:    payloadTemplate,
		headers:            headers,
	}
}

type WebhookListener struct {
	name               string
	Uri                string
	Log                *zap.SugaredLogger
	HttpClient         *http.Client
	events             []testkube.EventType
	selector           string
	payloadObjectField string
	payloadTemplate    string
	headers            map[string]string
}

func (l *WebhookListener) Name() string {
	return common.ListenerName(l.name)
}

func (l *WebhookListener) Selector() string {
	return l.selector
}

func (l *WebhookListener) Events() []testkube.EventType {
	return l.events
}
func (l *WebhookListener) Metadata() map[string]string {
	return map[string]string{
		"name":               l.Name(),
		"uri":                l.Uri,
		"selector":           l.selector,
		"events":             fmt.Sprintf("%v", l.events),
		"payloadObjectField": l.payloadObjectField,
		"payloadTemplate":    l.payloadTemplate,
		"headers":            fmt.Sprintf("%v", l.headers),
	}
}

func (l *WebhookListener) PayloadObjectField() string {
	return l.payloadObjectField
}

func (l *WebhookListener) PayloadTemplate() string {
	return l.payloadTemplate
}

func (l *WebhookListener) Headers() map[string]string {
	return l.headers
}

func (l *WebhookListener) Notify(event testkube.Event) (result testkube.EventResult) {
	body := bytes.NewBuffer([]byte{})
	log := l.Log.With(event.Log()...)

	var err error
	if l.payloadTemplate != "" {
		var tmpl *template.Template
		tmpl, err = template.New("webhook").Parse(l.payloadTemplate)
		if err != nil {
			log.Errorw("creating webhook template error", "error", err)
			return testkube.NewFailedEventResult(event.Id, err)
		}

		var buffer bytes.Buffer
		if err = tmpl.ExecuteTemplate(&buffer, "webhook", event); err != nil {
			log.Errorw("executing webhook template error", "error", err)
			return testkube.NewFailedEventResult(event.Id, err)
		}

		_, err = body.Write(buffer.Bytes())
	} else {
		err = json.NewEncoder(body).Encode(event)
		if err == nil && l.payloadObjectField != "" {
			data := map[string]string{l.payloadObjectField: string(body.Bytes())}
			body.Reset()
			err = json.NewEncoder(body).Encode(data)
		}
	}

	if err != nil {
		err = errors.Wrap(err, "webhook send json encode error")
		log.Errorw("webhook send json encode error", "error", err)
		return testkube.NewFailedEventResult(event.Id, err)
	}

	request, err := http.NewRequest(http.MethodPost, l.Uri, body)
	if err != nil {
		log.Errorw("webhook request creating error", "error", err)
		return testkube.NewFailedEventResult(event.Id, err)
	}

	request.Header.Set("Content-Type", "application/json")
	for key, value := range l.headers {
		request.Header.Set(key, value)
	}

	resp, err := l.HttpClient.Do(request)
	if err != nil {
		log.Errorw("webhook send error", "error", err)
		return testkube.NewFailedEventResult(event.Id, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorw("webhook read response error", "error", err)
		return testkube.NewFailedEventResult(event.Id, err)
	}

	responseStr := string(data)

	if resp.StatusCode >= 400 {
		err := fmt.Errorf("webhook response with bad status code: %d", resp.StatusCode)
		log.Errorw("webhook send error", "error", err, "status", resp.StatusCode)
		return testkube.NewFailedEventResult(event.Id, err).WithResult(responseStr)
	}

	log.Debugw("got webhook send result", "response", responseStr)
	return testkube.NewSuccessEventResult(event.Id, responseStr)
}

func (l *WebhookListener) Kind() string {
	return "webhook"
}
