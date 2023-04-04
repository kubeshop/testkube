package slack

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/slack"
)

var _ common.Listener = (*SlackListener)(nil)

func NewSlackListener(name, selector string, events []testkube.EventType, notifier *slack.Notifier) *SlackListener {
	return &SlackListener{
		name:          name,
		Log:           log.DefaultLogger,
		selector:      selector,
		events:        events,
		slackNotifier: notifier,
	}
}

type SlackListener struct {
	name          string
	Log           *zap.SugaredLogger
	events        []testkube.EventType
	selector      string
	slackNotifier *slack.Notifier
}

func (l *SlackListener) Name() string {
	return l.name
}

func (l *SlackListener) Selector() string {
	return l.selector
}

func (l *SlackListener) Events() []testkube.EventType {
	return l.events
}
func (l *SlackListener) Metadata() map[string]string {
	return map[string]string{
		"name":     l.Name(),
		"events":   fmt.Sprintf("%v", l.Events()),
		"selector": l.Selector(),
	}
}

func (l *SlackListener) Notify(event testkube.Event) (result testkube.EventResult) {
	err := l.slackNotifier.SendEvent(&event)
	if err != nil {
		return testkube.NewFailedEventResult(event.Id, err)
	}

	return testkube.NewSuccessEventResult(event.Id, "event sent to slack")
}

func (l *SlackListener) Kind() string {
	return "slack"
}
