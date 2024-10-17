package watchers

import (
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type jobEvents struct {
	events []*corev1.Event
}

type JobEvents interface {
	Original() []*corev1.Event
	Len() int
	Namespace() string
	PodName() string
	PodCreationTimestamp() time.Time
	PodDeletionTimestamp() time.Time
	FirstTimestamp() time.Time
	LastTimestamp() time.Time
	Finished() bool
	FinishTimestamp() time.Time
	Success() bool

	Error() bool
	ErrorReason() string
	ErrorMessage() string
}

func NewJobEvents(events []*corev1.Event) JobEvents {
	return &jobEvents{events: events}
}

func (j *jobEvents) Original() []*corev1.Event {
	return j.events
}

func (j *jobEvents) Len() int {
	return len(j.events)
}

func (j *jobEvents) Namespace() string {
	if len(j.events) == 0 {
		return ""
	}
	return j.events[0].Namespace
}

func (j *jobEvents) PodName() string {
	for i := range j.events {
		if j.events[i].Reason == "SuccessfulCreate" || j.events[i].Reason == "SuccessfulDelete" {
			// (SuccessfulCreate) Created pod: 66c49ca3284bce9380023421-78fmp
			// (SuccessfulDelete) Deleted pod: 66cc646a37732ad77e2ed368-ck9n7
			return j.events[i].Message[strings.LastIndex(j.events[i].Message, " ")+1:]
		}
	}
	return ""
}

func (j *jobEvents) PodCreationTimestamp() time.Time {
	for i := range j.events {
		if j.events[i].Reason == "SuccessfulCreate" {
			// (SuccessfulCreate) Created pod: 66c49ca3284bce9380023421-78fmp
			return GetEventTimestamp(j.events[i])
		}
	}
	return time.Time{}
}

func (j *jobEvents) PodDeletionTimestamp() time.Time {
	for i := range j.events {
		if j.events[i].Reason == "SuccessfulDelete" {
			// (SuccessfulDelete) Deleted pod: 66cc646a37732ad77e2ed368-ck9n7
			return GetEventTimestamp(j.events[i])
		}
	}
	return time.Time{}
}

func (j *jobEvents) FirstTimestamp() (ts time.Time) {
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

func (j *jobEvents) LastTimestamp() (ts time.Time) {
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

func (j *jobEvents) Finished() bool {
	return j.Success() || j.Error()
}

func (j *jobEvents) FinishTimestamp() time.Time {
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

func (j *jobEvents) Success() bool {
	for i := range j.events {
		if j.events[i].Reason == "Completed" {
			// (Completed) Job completed
			return true
		}
	}
	return false
}

func (j *jobEvents) Error() bool {
	for i := range j.events {
		if j.events[i].Reason == "BackoffLimitExceeded" || j.events[i].Reason == "DeadlineExceeded" {
			// (BackoffLimitExceeded) Job has reached the specified backoff limit
			// (DeadlineExceeded) Job was active longer than specified deadline
			return true
		}
	}
	return false
}

func (j *jobEvents) ErrorReason() string {
	for i := range j.events {
		if j.events[i].Reason == "BackoffLimitExceeded" || j.events[i].Reason == "DeadlineExceeded" {
			// (BackoffLimitExceeded) Job has reached the specified backoff limit
			// (DeadlineExceeded) Job was active longer than specified deadline
			return j.events[i].Reason
		}
	}
	return ""
}

func (j *jobEvents) ErrorMessage() string {
	for i := range j.events {
		if j.events[i].Reason == "BackoffLimitExceeded" || j.events[i].Reason == "DeadlineExceeded" {
			// (BackoffLimitExceeded) Job has reached the specified backoff limit
			// (DeadlineExceeded) Job was active longer than specified deadline
			return j.events[i].Message
		}
	}
	return ""
}
