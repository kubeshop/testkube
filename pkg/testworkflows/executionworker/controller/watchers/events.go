package watchers

import (
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type Events interface {
	Original() []*corev1.Event
	Len() int
	FirstTimestamp() time.Time
	LastTimestamp() time.Time
	FinishTimestamp() time.Time

	Error() bool
	ErrorReason() string
	ErrorMessage() string
	Debug() string
}

type joinedEvents struct {
	jobEvents JobEvents
	podEvents PodEvents
}

func (j *joinedEvents) Original() []*corev1.Event {
	result := append(append(make([]*corev1.Event, 0), j.jobEvents.Original()...), j.podEvents.Original()...)
	slices.SortFunc(result, func(a, b *corev1.Event) int {
		if a.FirstTimestamp.Time.Equal(b.FirstTimestamp.Time) {
			return 0
		} else if a.FirstTimestamp.Time.Before(b.FirstTimestamp.Time) {
			return -1
		} else {
			return 1
		}
	})
	return result
}

func (j *joinedEvents) Len() int {
	return j.jobEvents.Len() + j.podEvents.Len()
}

func (j *joinedEvents) FirstTimestamp() time.Time {
	ts1, ts2 := j.jobEvents.FirstTimestamp(), j.podEvents.FirstTimestamp()
	if ts1.Before(ts2) {
		return ts1
	}
	return ts2
}

func (j *joinedEvents) LastTimestamp() time.Time {
	ts1, ts2 := j.jobEvents.LastTimestamp(), j.podEvents.LastTimestamp()
	if ts1.After(ts2) {
		return ts1
	}
	return ts2
}

func (j *joinedEvents) FinishTimestamp() time.Time {
	ts1, ts2 := j.jobEvents.FinishTimestamp(), j.podEvents.FinishTimestamp()
	if ts1.Before(ts2) {
		return ts1
	}
	return ts2
}

func (j *joinedEvents) Error() bool {
	return j.jobEvents.Error() || j.podEvents.Error()
}

func (j *joinedEvents) ErrorReason() string {
	if j.podEvents.ErrorReason() != "" {
		return j.podEvents.ErrorReason()
	}
	return j.jobEvents.ErrorReason()
}

func (j *joinedEvents) ErrorMessage() string {
	if j.podEvents.ErrorMessage() != "" {
		return j.podEvents.ErrorMessage()
	}
	return j.jobEvents.ErrorMessage()
}

func (j *joinedEvents) Debug() string {
	return j.jobEvents.Debug() + "\n" + j.podEvents.Debug()
}
