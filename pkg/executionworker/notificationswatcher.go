package executionworker

import (
	"sync/atomic"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type notificationsWatcher struct {
	ch       chan testkube.TestWorkflowExecutionNotification
	finished atomic.Bool
	err      atomic.Value
}

func newNotificationsWatcher() *notificationsWatcher {
	return &notificationsWatcher{
		ch: make(chan testkube.TestWorkflowExecutionNotification),
	}
}

func (n *notificationsWatcher) send(notification testkube.TestWorkflowExecutionNotification) {
	n.ch <- notification
}

func (n *notificationsWatcher) close(err error) {
	if n.finished.CompareAndSwap(false, true) {
		if err != nil {
			n.err.Store(err)
		}
		close(n.ch)
	}
}

func (n *notificationsWatcher) Channel() <-chan testkube.TestWorkflowExecutionNotification {
	return n.ch
}

func (n *notificationsWatcher) Err() error {
	err := n.err.Load()
	if err == nil {
		return nil
	}
	return err.(error)
}

type NotificationsWatcher interface {
	Channel() <-chan testkube.TestWorkflowExecutionNotification
	Err() error
}
