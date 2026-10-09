package localinstall

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
	var images []string
	for _, c := range p.Status.ContainerStatuses {
		images = append(images, c.Image)
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
	st.image = strings.Join(images, ",")
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

// Second layer: a crash log can still print a secret.
type redactor struct {
	values []string
}

// Short values like "password" would mangle ordinary words.
func newRedactor(secrets ...string) redactor {
	var r redactor
	for _, v := range secrets {
		if len(v) < 12 {
			continue
		}
		r.values = append(r.values, v, url.QueryEscape(v), base64.StdEncoding.EncodeToString([]byte(v)))
	}
	return r
}

var secretPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	// Greedy up to the host, so an @ inside the password goes too.
	{regexp.MustCompile(`(://[^/:@\s]+:)[^\s/]+@`), "${1}[removed]@"},
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`), "${1}[removed]"},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), "[removed]"},
	{regexp.MustCompile(`(?i)((?:password|secret|token|api_?key|license)[\w-]*["']?\s*[:=]\s*["']?)[^\s"',]+`), "${1}[removed]"},
}

func (r redactor) clean(s string) string {
	for _, v := range r.values {
		s = strings.ReplaceAll(s, v, "[removed]")
	}
	for _, p := range secretPatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}
	return s
}

func renderReport(r report, rd redactor) string {
	var b strings.Builder
	r.write(&b)
	return rd.clean(b.String())
}

const (
	maxReportBytes = 1 << 20
	keptReports    = 10
	reportLogLines = 50
)

// Best effort: a failed save must not hide the install's error.
func (i *Installer) saveReport(since time.Time, installErr error, secrets ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	data, err := i.read(ctx)
	if err != nil {
		return "", err
	}
	snap, err := parseSnapshot(data)
	if err != nil {
		return "", err
	}
	var stuck *Stuck
	var se StuckError
	if errors.As(installErr, &se) {
		stuck = &se.Stuck
	} else if st, ok := findStuck(snap, since, time.Now()); ok {
		stuck = &st
	}
	r := buildReport(snap, stuck)
	for _, p := range r.services {
		if !p.ready && p.phase != string(corev1.PodSucceeded) {
			if out, err := i.logs(ctx, p.pod, false, reportLogLines); err == nil && len(out) > 0 {
				r.logs[p.pod] = cutLines(string(out))
			}
		}
	}
	now := time.Now()
	text := fmt.Sprintf("Testkube install report, saved after a failed install on %s.\n"+
		"Your license key and Testkube passwords are masked.\n\nError: %s\n\n%s",
		now.Format("2 Jan 2006 at 15:04"), installErr, renderReport(r, newRedactor()))
	text = capReport(newRedactor(secrets...).clean(text))
	dir := filepath.Join(i.dir, "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "install-"+now.Format("20060102-150405")+".txt")
	if err := writePrivate(path, []byte(text)); err != nil {
		return "", err
	}
	pruneReports(dir)
	return path, nil
}

// Names sort by time, so the oldest come first.
func pruneReports(dir string) {
	old, _ := filepath.Glob(filepath.Join(dir, "install-*.txt"))
	sort.Strings(old)
	for len(old) > keptReports {
		_ = os.Remove(old[0])
		old = old[1:]
	}
}

// Cut after masking, so a cut can't leave half a secret.
func capReport(text string) string {
	if len(text) <= maxReportBytes {
		return text
	}
	return text[:maxReportBytes] + "\n[cut at 1 MB]\n"
}

func cutLines(out string) []string {
	all := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var lines []string
	for _, line := range all[max(0, len(all)-reportLogLines):] {
		if len(line) > 2048 {
			line = line[:2048] + "…"
		}
		lines = append(lines, line)
	}
	return lines
}
