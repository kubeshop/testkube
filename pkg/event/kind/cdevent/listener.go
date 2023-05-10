package cdevent

import (
	"context"
	"fmt"

	cdevents "github.com/cdevents/sdk-go/pkg/api"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	cde "github.com/kubeshop/testkube/pkg/mapper/cdevents"
	"github.com/kubeshop/testkube/pkg/version"
)

var _ common.Listener = (*CDEventListener)(nil)

func NewCDEventListener(name, selector, clusterID string, events []testkube.EventType, client cloudevents.Client) *CDEventListener {
	return &CDEventListener{
		name:       name,
		Log:        log.DefaultLogger,
		selector:   selector,
		events:     events,
		client:     client,
		clusterID:  clusterID,
		appVersion: "testkube-api:" + version.Version,
	}
}

type CDEventListener struct {
	name       string
	Log        *zap.SugaredLogger
	events     []testkube.EventType
	selector   string
	client     cloudevents.Client
	clusterID  string
	appVersion string
}

func (l *CDEventListener) Name() string {
	return l.name
}

func (l *CDEventListener) Selector() string {
	return l.selector
}

func (l *CDEventListener) Events() []testkube.EventType {
	return l.events
}
func (l *CDEventListener) Metadata() map[string]string {
	return map[string]string{
		"name":     l.Name(),
		"events":   fmt.Sprintf("%v", l.Events()),
		"selector": l.Selector(),
	}
}

func (l *CDEventListener) Notify(event testkube.Event) (result testkube.EventResult) {
	// Create the base event
	ev, err := cde.MapTestkubeEventToCDEvent(event, l.clusterID, l.appVersion)
	if err != nil {
		return testkube.NewFailedEventResult(event.Id, err)
	}

	ce, err := cdevents.AsCloudEvent(ev)
	if result := l.client.Send(context.Background(), *ce); cloudevents.IsUndelivered(result) {
		return testkube.NewFailedEventResult(event.Id, fmt.Errorf("failed to send, %v", result))
	}

	return testkube.NewSuccessEventResult(event.Id, "event sent to cd event")
}

func (l *CDEventListener) Kind() string {
	return "cdevent"
}
