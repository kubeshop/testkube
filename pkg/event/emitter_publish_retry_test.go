package event

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
)

// flakyBus mimics NATS returning a transient error N times before the publish
// eventually goes through. Used to prove the emitter now retries transient
// PublishTopic failures instead of dropping the event.
type flakyBus struct {
	*bus.EventBusMock
	failuresBeforeSuccess int
	calls                 int32
}

func (f *flakyBus) PublishTopic(topic string, event testkube.Event) error {
	n := atomic.AddInt32(&f.calls, 1)
	if int(n) <= f.failuresBeforeSuccess {
		return errors.New("nats connection refused")
	}
	return f.EventBusMock.PublishTopic(topic, event)
}

// alwaysFailingBus never returns success, letting the retry loop exhaust its
// budget so we can assert the final failure surfaces cleanly.
type alwaysFailingBus struct {
	*bus.EventBusMock
	calls int32
}

func (f *alwaysFailingBus) PublishTopic(topic string, event testkube.Event) error {
	atomic.AddInt32(&f.calls, 1)
	return errors.New("nats connection refused")
}

func TestEmitter_Notify_RetriesTransientPublishThenSucceeds(t *testing.T) {
	prev := publishRetryBaseDelay
	publishRetryBaseDelay = time.Millisecond
	t.Cleanup(func() { publishRetryBaseDelay = prev })

	fb := &flakyBus{EventBusMock: bus.NewEventBusMock(), failuresBeforeSuccess: 2}
	emitter := NewEmitter(fb, nil, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
	defer emitter.eventCache.Stop()

	evt := testkube.Event{Id: "e1", Type_: testkube.EventStartTestWorkflow}
	emitter.Notify(evt)

	assert.Equal(t, int32(3), atomic.LoadInt32(&fb.calls))
}

func TestEmitter_Notify_GivesUpAfterRetryBudget(t *testing.T) {
	prev := publishRetryBaseDelay
	publishRetryBaseDelay = time.Millisecond
	t.Cleanup(func() { publishRetryBaseDelay = prev })

	fb := &alwaysFailingBus{EventBusMock: bus.NewEventBusMock()}
	emitter := NewEmitter(fb, nil, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
	defer emitter.eventCache.Stop()

	evt := testkube.Event{Id: "e2", Type_: testkube.EventStartTestWorkflow}
	emitter.Notify(evt)

	assert.Equal(t, int32(publishRetryCount), atomic.LoadInt32(&fb.calls))
}
