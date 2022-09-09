package slack

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/slacknotifier"
	"go.uber.org/zap"
)

var _ common.Listener = &SlackListener{}

func NewSlackListener(name, selector string, events []testkube.EventType) *SlackListener {
	return &SlackListener{
		name:     name,
		Log:      log.DefaultLogger,
		selector: selector,
		events:   events,
	}
}

type SlackListener struct {
	name     string
	Log      *zap.SugaredLogger
	events   []testkube.EventType
	selector string
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
	return map[string]string{}
}

func (l *SlackListener) Notify(event testkube.Event) (result testkube.EventResult) {
	err := slacknotifier.SendEvent(event.Type_, *event.Execution)
	if err != nil {
		return testkube.NewFailedEventResult(event.Id, err)
	}

	return testkube.NewSuccessEventResult(event.Id, "event sent to slack")
}

func (l *SlackListener) Kind() string {
	return "slack"
}
