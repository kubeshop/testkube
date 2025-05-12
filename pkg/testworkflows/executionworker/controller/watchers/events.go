package watchers

import (
	"fmt"
	"slices"
	"strings"
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

type bareEvents struct {
	events []*corev1.Event
}

func NewEvents(events []*corev1.Event) Events {
	return &bareEvents{events}
}

func (j *bareEvents) Original() []*corev1.Event {
	return j.events
}

func (j *bareEvents) Len() int {
	return len(j.events)
}

func (j *bareEvents) FirstTimestamp() (ts time.Time) {
	for i := range j.events {
		if ts.IsZero() || ts.After(j.events[i].CreationTimestamp.Time) {
			ts = j.events[i].CreationTimestamp.Time
		}
		if ts.IsZero() || ts.After(j.events[i].FirstTimestamp.Time) {
			ts = j.events[i].FirstTimestamp.Time
		}
		if ts.IsZero() || ts.After(j.events[i].LastTimestamp.Time) {
			ts = j.events[i].LastTimestamp.Time
		}
	}
	return ts
}

func (j *bareEvents) LastTimestamp() (ts time.Time) {
	for i := range j.events {
		if ts.Before(j.events[i].CreationTimestamp.Time) {
			ts = j.events[i].CreationTimestamp.Time
		}
		if ts.Before(j.events[i].FirstTimestamp.Time) {
			ts = j.events[i].FirstTimestamp.Time
		}
		if ts.Before(j.events[i].LastTimestamp.Time) {
			ts = j.events[i].LastTimestamp.Time
		}
	}
	return ts
}

func (j *bareEvents) Finished() bool {
	return j.Success() || j.Error()
}

func (j *bareEvents) FinishTimestamp() time.Time {
	for i := range j.events {
		if j.events[i].Reason == "BackoffLimitExceeded" || j.events[i].Reason == "DeadlineExceeded" || j.events[i].Reason == "Completed" {
			// (BackoffLimitExceeded) Job has reached the specified backoff limit
			// (DeadlineExceeded) Job was active longer than specified deadline
			// (Completed) Job completed
			return GetEventTimestamp(j.events[i])
		}
	}
	return time.Time{}
}

func (j *bareEvents) Success() bool {
	for i := range j.events {
		if j.events[i].Reason == "Completed" {
			// (Completed) Job completed
			return true
		}
	}
	return false
}

func (j *bareEvents) Error() bool {
	return false // FIXME
}

func (j *bareEvents) ErrorReason() string {
	return "" // FIXME
}

func (j *bareEvents) ErrorMessage() string {
	return "" // FIXME
}

func (j *bareEvents) Debug() string {
	firstTs := j.FirstTimestamp()
	result := make([]string, len(j.events))
	for i := range j.events {
		result[i] = fmt.Sprintf("[%.1fs] %s: %s", float64(GetEventTimestamp(j.events[i]).Sub(firstTs))/float64(time.Second), j.events[i].Reason, j.events[i].Message)
	}

	return strings.Join(result, ", ")
}
