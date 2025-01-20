package executionworkertypes

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

func NewNotificationsWatcher() WritableNotificationsWatcher {
	return channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
}

type NotificationsWatcher interface {
	Channel() <-chan *testkube.TestWorkflowExecutionNotification
	All() ([]*testkube.TestWorkflowExecutionNotification, error)
	Err() error
}

type WritableNotificationsWatcher interface {
	NotificationsWatcher
	Send(notification *testkube.TestWorkflowExecutionNotification)
	Close(err error)
}

func NewStatusNotificationsWatcher() WritableStatusNotificationsWatcher {
	return channels.NewWatcher[StatusNotification]()
}

type StatusNotificationsWatcher interface {
	Channel() <-chan StatusNotification
	All() ([]StatusNotification, error)
	Err() error
}

type WritableStatusNotificationsWatcher interface {
	StatusNotificationsWatcher
	Send(notification StatusNotification)
	Close(err error)
}
