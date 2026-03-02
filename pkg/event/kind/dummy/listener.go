package dummy

import (
	"sync"
	"sync/atomic"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

var _ common.Listener = (*DummyListener)(nil)

type DummyListener struct {
	Id                 string
	NotificationCount  int32
	SelectorString     string
	Types              []testkube.EventType
	ReceivedEventTypes []testkube.EventType
	mu                 sync.Mutex
}

func (l *DummyListener) GetNotificationCount() int {
	cnt := atomic.LoadInt32(&l.NotificationCount)
	return int(cnt)
}

func (l *DummyListener) GetReceivedEventTypes() []testkube.EventType {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]testkube.EventType, len(l.ReceivedEventTypes))
	copy(result, l.ReceivedEventTypes)
	return result
}

func (l *DummyListener) Match(event testkube.Event) bool {
	_, valid := event.Valid(l.Group(), l.Selector(), l.Events())
	return valid
}

func (l *DummyListener) Notify(event testkube.Event) testkube.EventResult {
	log.DefaultLogger.Infow("DummyListener notified", "listenerId", l.Id, "event", event)
	atomic.AddInt32(&l.NotificationCount, 1)

	// Track received event types for testing
	l.mu.Lock()
	if event.Type_ != nil {
		l.ReceivedEventTypes = append(l.ReceivedEventTypes, *event.Type_)
	}
	l.mu.Unlock()

	return testkube.EventResult{Id: event.Id}
}

func (l *DummyListener) Name() string {
	if l.Id != "" {
		return l.Id
	}
	return "dummy"
}

func (l *DummyListener) Events() []testkube.EventType {
	if l.Types != nil {
		return l.Types
	}

	return testkube.AllEventTypes
}

func (l *DummyListener) Selector() string {
	return l.SelectorString
}

func (l *DummyListener) Kind() string {
	return "dummy"
}

func (l *DummyListener) Group() string {
	return ""
}

func (l *DummyListener) Metadata() map[string]string {
	return map[string]string{
		"id":       l.Name(),
		"selector": l.Selector(),
	}
}
