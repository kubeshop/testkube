package watchers

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

type containerEvents struct {
	events []*corev1.Event
}

type ContainerEvents interface {
	Original() []*corev1.Event
	Len() int
	Created() bool
	Started() bool
	CreationTimestamp() time.Time
	StartTimestamp() time.Time
}

func NewContainerEvents(events []*corev1.Event) ContainerEvents {
	return &containerEvents{events: events}
}

func (c *containerEvents) Original() []*corev1.Event {
	return c.events
}

func (c *containerEvents) Len() int {
	return len(c.events)
}

func (c *containerEvents) Created() bool {
	for i := range c.events {
		if c.events[i].Reason == "Created" {
			// (Created) Created container 1
			return true
		}
	}
	return false
}

func (c *containerEvents) Started() bool {
	for i := range c.events {
		if c.events[i].Reason == "Started" {
			// (Started) Started container 1
			return true
		}
	}
	return false
}

func (c *containerEvents) CreationTimestamp() time.Time {
	for i := range c.events {
		if c.events[i].Reason == "Created" {
			// (Created) Created container 1
			return GetEventTimestamp(c.events[i])
		}
	}
	return time.Time{}
}

func (c *containerEvents) StartTimestamp() time.Time {
	for i := range c.events {
		if c.events[i].Reason == "Started" {
			// (Started) Started container 1
			return GetEventTimestamp(c.events[i])
		}
	}
	return time.Time{}
}
