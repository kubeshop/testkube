package dummy

import (
	"sync/atomic"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

var _ common.Listener = (*DummyListener)(nil)

type DummyListener struct {
	Id                string
	NotificationCount int32
	SelectorString    string
}

func (l *DummyListener) GetNotificationCount() int {
	cnt := atomic.LoadInt32(&l.NotificationCount)
	return int(cnt)
}

func (l *DummyListener) Notify(event testkube.Event) testkube.EventResult {
	log.DefaultLogger.Infow("DummyListener notified", "listenerId", l.Id, "event", event)
	atomic.AddInt32(&l.NotificationCount, 1)
	return testkube.EventResult{Id: event.Id}
}

func (l *DummyListener) Name() string {
	if l.Id != "" {
		return l.Id
	}
	return "dummy"
}

func (l *DummyListener) Events() []testkube.EventType {
	return testkube.AllEventTypes
}

func (l *DummyListener) Selector() string {
	return l.SelectorString
}

func (l *DummyListener) Kind() string {
	return "dummy"
}

func (l *DummyListener) Metadata() map[string]string {
	return map[string]string{
		"id":       l.Name(),
		"selector": l.Selector(),
	}
}
