package v1

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

func TestStreamableWorkflowNotificationsAppliesResumeCursorAndSequencing(t *testing.T) {
	watcher := executionworkertypes.NewNotificationsWatcher()
	go func() {
		watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "temporary", Temporary: true})
		watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "first"})
		watcher.Send(&testkube.TestWorkflowExecutionNotification{Result: &testkube.TestWorkflowResult{}})
		watcher.Close(nil)
	}()

	var notifications []testkube.TestWorkflowExecutionNotification
	for notification := range streamableWorkflowNotifications(watcher.Channel(), 1) {
		notifications = append(notifications, notification)
	}

	require.Len(t, notifications, 2)
	assert.Zero(t, notifications[0].SeqNo)
	assert.True(t, notifications[0].Temporary)
	assert.Equal(t, "log", notifications[0].EventType)
	assert.Equal(t, "temporary", notifications[0].Log)

	assert.Equal(t, int32(2), notifications[1].SeqNo)
	assert.Equal(t, "result", notifications[1].EventType)
	require.NotNil(t, notifications[1].Result)
}

func TestStreamableWorkflowNotificationsSendsHeartbeatWhileQuiet(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	source := make(chan *testkube.TestWorkflowExecutionNotification)

	notifications := streamableWorkflowNotificationsWithHeartbeat(done, source, 0, 5*time.Millisecond)

	select {
	case notification := <-notifications:
		assert.Equal(t, "heartbeat", notification.EventType)
		assert.Zero(t, notification.SeqNo)
		assert.Empty(t, notification.Log)
		assert.Nil(t, notification.Output)
		assert.Nil(t, notification.Result)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected heartbeat while workflow notification stream is quiet")
	}
}

func TestStreamableWorkflowNotificationsHeartbeatDoesNotAdvanceSeqNo(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	source := make(chan *testkube.TestWorkflowExecutionNotification, 1)

	notifications := streamableWorkflowNotificationsWithHeartbeat(done, source, 0, 5*time.Millisecond)

	select {
	case notification := <-notifications:
		require.Equal(t, "heartbeat", notification.EventType)
		assert.Zero(t, notification.SeqNo)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected initial heartbeat")
	}

	source <- &testkube.TestWorkflowExecutionNotification{Log: "after quiet period"}

	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case notification := <-notifications:
			if notification.EventType == "heartbeat" {
				continue
			}
			assert.Equal(t, "log", notification.EventType)
			assert.Equal(t, int32(1), notification.SeqNo)
			assert.Equal(t, "after quiet period", notification.Log)
			return
		case <-timeout:
			t.Fatal("expected durable log notification after heartbeat")
		}
	}
}

func TestWorkflowNotificationEventType(t *testing.T) {
	assert.Equal(t, "log", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{Log: "hello"}))
	assert.Equal(t, "result", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{Result: &testkube.TestWorkflowResult{}}))
	assert.Equal(t, "output", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{Output: &testkube.TestWorkflowOutput{Name: "out"}}))
	assert.Equal(t, "", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{}))
}

func TestWorkflowNotificationResumableIgnoresTemporaryNotifications(t *testing.T) {
	assert.False(t, workflowNotificationResumable(testkube.TestWorkflowExecutionNotification{Log: "temporary", Temporary: true}))
	assert.True(t, workflowNotificationResumable(testkube.TestWorkflowExecutionNotification{Log: "durable", Ts: time.Now()}))
}

func TestWriteWorkflowNotificationEventWritesServerSentEvent(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	err := writeWorkflowNotificationEvent(w, json.NewEncoder(w), testkube.TestWorkflowExecutionNotification{SeqNo: 3, EventType: "log", Log: "line"})

	require.NoError(t, err)
	assert.Equal(t, "id: 3\nevent: log\ndata: {\"ts\":\"0001-01-01T00:00:00Z\",\"seqNo\":3,\"eventType\":\"log\",\"log\":\"line\"}\n\n", buf.String())
}

type closedConnWriter struct{}

func (closedConnWriter) Write([]byte) (int, error) { return 0, errors.New("connection closed") }

func TestWriteWorkflowNotificationEventReportsClosedStream(t *testing.T) {
	w := bufio.NewWriterSize(closedConnWriter{}, 16)

	err := writeWorkflowNotificationEvent(w, json.NewEncoder(w), testkube.TestWorkflowExecutionNotification{Log: "a line longer than the 16 byte buffer"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection closed")
}

// fakeNotificationsWatcher feeds notifications until the watch context is cancelled, counting what was sent.
type fakeNotificationsWatcher struct {
	ch   chan *testkube.TestWorkflowExecutionNotification
	sent atomic.Int32
}

func newFakeNotificationsWatcher(ctx context.Context, total int) *fakeNotificationsWatcher {
	f := &fakeNotificationsWatcher{ch: make(chan *testkube.TestWorkflowExecutionNotification)}
	go func() {
		defer close(f.ch)
		for i := 0; i < total; i++ {
			select {
			case f.ch <- &testkube.TestWorkflowExecutionNotification{Log: "line"}:
				f.sent.Add(1)
			case <-ctx.Done():
				return
			}
		}
	}()
	return f
}

func (f *fakeNotificationsWatcher) Channel() <-chan *testkube.TestWorkflowExecutionNotification {
	return f.ch
}
func (f *fakeNotificationsWatcher) All() ([]*testkube.TestWorkflowExecutionNotification, error) {
	return nil, nil
}
func (f *fakeNotificationsWatcher) Err() error { return nil }

func TestWriteNotificationStreamStopsAndReleasesWatcherWhenClientIsGone(t *testing.T) {
	s := &TestkubeAPI{Log: log.DefaultLogger}
	watchCtx, stop := context.WithCancel(context.Background())
	defer stop()
	const total = 1000
	source := newFakeNotificationsWatcher(watchCtx, total)

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.writeNotificationStream(bufio.NewWriter(closedConnWriter{}), watchCtx, stop, "id", source, 0, time.Hour)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("stream kept writing after the client was gone")
	}
	assert.Error(t, watchCtx.Err(), "watcher context should be cancelled")
	assert.Less(t, int(source.sent.Load()), total, "notifications should not be drained into a closed stream")
}

func TestWriteNotificationStreamWritesAllNotificationsAndReleasesWatcher(t *testing.T) {
	s := &TestkubeAPI{Log: log.DefaultLogger}
	watchCtx, stop := context.WithCancel(context.Background())
	defer stop()
	source := newFakeNotificationsWatcher(watchCtx, 3)
	var buf bytes.Buffer

	s.writeNotificationStream(bufio.NewWriter(&buf), watchCtx, stop, "id", source, 0, time.Hour)

	assert.Equal(t, 3, strings.Count(buf.String(), "data: "))
	assert.Error(t, watchCtx.Err(), "watcher context should be cancelled when the stream ends")
}
