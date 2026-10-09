package localinstall

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const maxReportEvents = 100

// Picked fields only: pod specs carry the license and keys.
type report struct {
	stuck    *Stuck
	services []serviceState
	events   []eventLine
	// Pod name to its last log lines.
	logs map[string][]string
}

type serviceState struct {
	pod, phase, image, reason, message string
	ready                              bool
	restarts, exitCode                 int32
}

type eventLine struct {
	at                            time.Time
	kind, reason, object, message string
	count                         int32
}

func buildReport(s clusterSnapshot, stuck *Stuck) report {
	r := report{stuck: stuck, logs: map[string][]string{}}
	for i := range s.pods {
		r.services = append(r.services, stateOf(&s.pods[i]))
	}
	sort.Slice(r.services, func(a, b int) bool { return r.services[a].pod < r.services[b].pod })
	for _, e := range s.events {
		r.events = append(r.events, eventLine{at: eventTime(e), kind: e.Type, reason: e.Reason,
			object: e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name, message: e.Message, count: e.Count})
	}
	sort.Slice(r.events, func(a, b int) bool { return r.events[a].at.After(r.events[b].at) })
	if len(r.events) > maxReportEvents {
		r.events = r.events[:maxReportEvents]
	}
	return r
}

func stateOf(p *corev1.Pod) serviceState {
	st := serviceState{pod: p.Name, phase: string(p.Status.Phase)}
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			st.ready = c.Status == corev1.ConditionTrue
		}
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse {
			st.reason, st.message = c.Reason, c.Message
		}
	}
	for _, c := range p.Status.ContainerStatuses {
		st.image = c.Image
		st.restarts = max(st.restarts, c.RestartCount)
		if w := c.State.Waiting; w != nil {
			st.reason, st.message = w.Reason, w.Message
		}
		if t := c.LastTerminationState.Terminated; t != nil {
			st.exitCode = t.ExitCode
			if st.reason == "" {
				st.reason = t.Reason
			}
		}
	}
	return st
}

func (r report) write(w io.Writer) {
	if r.stuck != nil {
		fmt.Fprintf(w, "Stuck: %s, reason %s %s\n\n", r.stuck.Service, r.stuck.Reason, r.stuck.Detail)
	}
	fmt.Fprintln(w, "Pods:")
	for _, s := range r.services {
		fmt.Fprintf(w, "  %s  phase=%s ready=%t restarts=%d exit=%d image=%s\n", s.pod, s.phase, s.ready, s.restarts, s.exitCode, s.image)
		switch {
		case s.message != "":
			fmt.Fprintf(w, "    %s: %s\n", s.reason, s.message)
		case s.reason != "":
			fmt.Fprintf(w, "    %s\n", s.reason)
		}
	}
	fmt.Fprintf(w, "\nEvents (newest %d):\n", maxReportEvents)
	for _, e := range r.events {
		fmt.Fprintf(w, "  %s  %s %s %s x%d: %s\n", e.at.UTC().Format(time.RFC3339), e.kind, e.reason, e.object, e.count, e.message)
	}
	pods := make([]string, 0, len(r.logs))
	for pod := range r.logs {
		pods = append(pods, pod)
	}
	sort.Strings(pods)
	for _, pod := range pods {
		fmt.Fprintf(w, "\nLast log lines of %s:\n  %s\n", pod, strings.Join(r.logs[pod], "\n  "))
	}
}
