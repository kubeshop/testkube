package cdevent

import (
	"context"
	"fmt"
	"strings"

	cdevents "github.com/cdevents/sdk-go/pkg/api"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	cde "github.com/kubeshop/testkube/pkg/mapper/cdevents"
)

var _ common.Listener = (*CDEventListener)(nil)

func NewCDEventListener(name, selector, clusterID, defaultNamespace, dashboardURI string, events []testkube.EventType, client cloudevents.Client) *CDEventListener {
	return &CDEventListener{
		name:             name,
		Log:              log.DefaultLogger,
		selector:         selector,
		events:           events,
		client:           client,
		clusterID:        clusterID,
		defaultNamespace: defaultNamespace,
		dashboardURI:     dashboardURI,
	}
}

type CDEventListener struct {
	name             string
	Log              *zap.SugaredLogger
	events           []testkube.EventType
	selector         string
	client           cloudevents.Client
	clusterID        string
	defaultNamespace string
	dashboardURI     string
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

func (l *CDEventListener) Match(event testkube.Event) bool {
	_, valid := event.Valid(l.Selector(), l.Events())
	return valid
}

func (l *CDEventListener) Notify(event testkube.Event) (result testkube.EventResult) {
	// Check if CDEvents are silenced for test workflow executions
	if event.TestWorkflowExecution != nil && event.TestWorkflowExecution.SilentMode != nil && event.TestWorkflowExecution.SilentMode.Cdevents {
		l.Log.With("event", event.Id).Debug("CDEvents silenced for test workflow execution")
		return testkube.NewSuccessEventResult(event.Id, "CDEvents silenced for test workflow execution")
	}

	// Create the base event
	namespace := l.defaultNamespace
	if event.TestExecution != nil {
		namespace = event.TestExecution.TestNamespace
	}

	ev, err := cde.MapTestkubeEventToCDEvent(event, l.clusterID, namespace, l.dashboardURI)
	if err != nil {
		return testkube.NewFailedEventResult(event.Id, err)
	}

	if err := l.sendCDEvent(ev); err != nil {
		return testkube.NewFailedEventResult(event.Id, err)
	}

	if event.Type_ != nil && (*event.Type_ == *testkube.EventEndTestWorkflowAborted || *event.Type_ == *testkube.EventEndTestWorkflowFailed ||
		*event.Type_ == *testkube.EventEndTestWorkflowSuccess || *event.Type_ == *testkube.EventEndTestWorkflowCanceled) {
		// Create the output event
		ev, err = cde.MapTestkubeTestWorkflowLogToCDEvent(event, l.clusterID, l.dashboardURI)
		if err != nil {
			return testkube.NewFailedEventResult(event.Id, err)
		}

		if err := l.sendCDEvent(ev); err != nil {
			return testkube.NewFailedEventResult(event.Id, err)
		}
	}

	return testkube.NewSuccessEventResult(event.Id, "event sent to cd event")
}

func (l *CDEventListener) Kind() string {
	return "cdevent"
}

func (l *CDEventListener) Group() string {
	return "default-group"
}

func (l *CDEventListener) sendCDEvent(ev cdevents.CDEventReader) error {
	ce, err := cdevents.AsCloudEvent(ev)
	if err != nil {
		return err
	}

	if result := l.client.Send(context.Background(), *ce); cloudevents.IsUndelivered(result) {
		return fmt.Errorf("failed to deliver, %v", result)
	} else if msg := result.Error(); msg != "" && !strings.Contains(msg, "200") {
		return fmt.Errorf("failed to send, %s", msg)
	}

	return nil
}
