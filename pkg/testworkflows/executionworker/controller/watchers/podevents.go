package watchers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

var (
	scheduledRe = regexp.MustCompile(`/\s+(\S+)`)
)

type podEvents struct {
	events []*corev1.Event
}

type PodEvents interface {
	Original() []*corev1.Event
	Len() int
	Name() string
	Namespace() string
	NodeName() string
	FirstTimestamp() time.Time
	LastTimestamp() time.Time
	StartTimestamp() time.Time
	FinishTimestamp() time.Time
	Error() bool
	ErrorReason() string
	ErrorMessage() string
	Debug() string

	Container(name string) ContainerEvents
}

func NewPodEvents(events []*corev1.Event) PodEvents {
	return &podEvents{events}
}

func (p *podEvents) Original() []*corev1.Event {
	return p.events
}

func (p *podEvents) Len() int {
	return len(p.events)
}

func (p *podEvents) Name() string {
	// Determine by involvedObject
	for i := range p.events {
		if p.events[i].InvolvedObject.Kind == "Pod" {
			return p.events[i].InvolvedObject.Name
		}
	}

	// Determine by message (fallback)
	for i := range p.events {
		if p.events[i].Reason == "Scheduled" {
			// (Scheduled) Successfully assigned distributed-tests/66c49ca3284bce9380023421-78fmp to homelab
			match := scheduledRe.FindStringSubmatch(p.events[i].Message)
			if match != nil {
				return match[1]
			}
		}
	}
	return ""
}

func (p *podEvents) Namespace() string {
	if len(p.events) == 0 {
		return ""
	}
	return p.events[0].Namespace
}

func (p *podEvents) NodeName() string {
	for i := range p.events {
		if p.events[i].Reason == "Scheduled" {
			// (Scheduled) Successfully assigned distributed-tests/66c49ca3284bce9380023421-78fmp to homelab
			return p.events[i].Message[strings.LastIndex(p.events[i].Message, " ")+1:]
		}
	}
	return ""
}

func (p *podEvents) FirstTimestamp() (ts time.Time) {
	for i := range p.events {
		if ts.IsZero() || ts.After(p.events[i].CreationTimestamp.Time) {
			ts = p.events[i].CreationTimestamp.Time
		}
		if ts.IsZero() || ts.After(p.events[i].FirstTimestamp.Time) {
			ts = p.events[i].FirstTimestamp.Time
		}
		if ts.IsZero() || ts.After(p.events[i].LastTimestamp.Time) {
			ts = p.events[i].LastTimestamp.Time
		}
	}
	return ts
}

func (p *podEvents) LastTimestamp() (ts time.Time) {
	for i := range p.events {
		if ts.Before(p.events[i].CreationTimestamp.Time) {
			ts = p.events[i].CreationTimestamp.Time
		}
		if ts.Before(p.events[i].FirstTimestamp.Time) {
			ts = p.events[i].FirstTimestamp.Time
		}
		if ts.Before(p.events[i].LastTimestamp.Time) {
			ts = p.events[i].LastTimestamp.Time
		}
	}
	return ts
}

func (p *podEvents) StartTimestamp() time.Time {
	for i := range p.events {
		if p.events[i].Reason == "Scheduled" {
			// (Scheduled) Successfully assigned distributed-tests/66c49ca3284bce9380023421-78fmp to homelab
			return GetEventTimestamp(p.events[i])
		}
	}
	return time.Time{}
}

func (p *podEvents) FinishTimestamp() time.Time {
	for i := range p.events {
		if p.events[i].Reason == "Evicted" || p.events[i].Reason == "ExceededGracePeriod" {
			// (Evicted) The node was low on resource: ephemeral-storage
			// (ExceededGracePeriod) Container runtime did not kill the pod within specified grace period
			return GetEventTimestamp(p.events[i])
		}

		// TODO: Consider approximation, but quite accurate
		// (Killing) Stopping container 1 [ONLY NUMERIC CONTAINERS]
		// (Preempting) ???
		// (Failed) Back-off restarting failed container [ONLY NUMERIC CONTAINERS]
	}
	return time.Time{}
}

func (p *podEvents) Error() bool {
	for i := range p.events {
		if p.events[i].Reason == "Evicted" || p.events[i].Reason == "ExceededGracePeriod" {
			// (Evicted) The node was low on resource: ephemeral-storage
			// (ExceededGracePeriod) Container runtime did not kill the pod within specified grace period
			return true
		}

		// TODO: Consider approximation, but quite accurate
		// (Killing) Stopping container 1 [ONLY NUMERIC CONTAINERS]
		// (Preempting) ???
		// (Failed) Back-off restarting failed container [ONLY NUMERIC CONTAINERS]
	}
	return false
}

func (p *podEvents) ErrorReason() string {
	for i := range p.events {
		if p.events[i].Reason == "Evicted" || p.events[i].Reason == "ExceededGracePeriod" {
			// (Evicted) The node was low on resource: ephemeral-storage
			// (ExceededGracePeriod) Container runtime did not kill the pod within specified grace period
			return p.events[i].Reason
		}

		// TODO: Consider approximation, but quite accurate
		// (Killing) Stopping container 1 [ONLY NUMERIC CONTAINERS]
		// (Preempting) ???
		// (Failed) Back-off restarting failed container [ONLY NUMERIC CONTAINERS]
	}
	return ""
}

func (p *podEvents) ErrorMessage() string {
	for i := range p.events {
		if p.events[i].Reason == "Evicted" || p.events[i].Reason == "ExceededGracePeriod" {
			// (Evicted) The node was low on resource: ephemeral-storage
			// (ExceededGracePeriod) Container runtime did not kill the pod within specified grace period
			return p.events[i].Message
		}

		// TODO: Consider approximation, but quite accurate
		// (Killing) Stopping container 1 [ONLY NUMERIC CONTAINERS]
		// (Preempting) ???
		// (Failed) Back-off restarting failed container [ONLY NUMERIC CONTAINERS]
	}
	return ""
}

func (p *podEvents) Debug() string {
	firstTs := p.FirstTimestamp()
	result := make([]string, len(p.events))
	for i := range p.events {
		result[i] = fmt.Sprintf("[%.1fs] %s: %s", float64(GetEventTimestamp(p.events[i]).Sub(firstTs))/float64(time.Second), p.events[i].Reason, p.events[i].Message)
	}
	return strings.Join(result, ", ")
}

func (p *podEvents) Container(name string) ContainerEvents {
	if name == "" {
		return nil
	}
	result := make([]*corev1.Event, 0)
	for i := range p.events {
		if GetEventContainerName(p.events[i]) == name {
			result = append(result, p.events[i])
		}
	}
	return NewContainerEvents(result)
}
