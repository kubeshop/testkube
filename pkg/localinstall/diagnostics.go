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
	"unicode/utf8"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

const maxReportEvents = 100

// Picked fields only: pod specs carry the license and keys.
type report struct {
	stuck     *Stuck
	services  []serviceState
	events    []eventLine
	logsByPod map[string][]string
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
	r := report{stuck: stuck, logsByPod: map[string][]string{}}
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
	pods := make([]string, 0, len(r.logsByPod))
	for pod := range r.logsByPod {
		pods = append(pods, pod)
	}
	sort.Strings(pods)
	for _, pod := range pods {
		fmt.Fprintf(w, "\nLast log lines of %s:\n  %s\n", pod, strings.Join(r.logsByPod[pod], "\n  "))
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
	// Greedy to the host: an @ in the password goes too.
	{regexp.MustCompile(`(://[^/:@\s]+:)[^\s/]+@`), "${1}[removed]@"},
	{regexp.MustCompile(`(?i)((?:bearer|basic)\s+)[A-Za-z0-9._~+/=-]+`), "${1}[removed]"},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), "[removed]"},
	{regexp.MustCompile(`(?i)((?:password|secret|token|api[_-]?key|license)[\w-]*["']?\s*[:=]\s*["']?)[^\s"',]+`), "${1}[removed]"},
	{regexp.MustCompile(`(?i)(--(?:password|secret|token|api[_-]?key|license)[\w-]*\s+)\S+`), "${1}[removed]"},
}

// From the demo values, so a chart bump keeps up.
func demoSecrets() []string {
	var demo map[string]any
	if yaml.Unmarshal(EnterpriseDemoValues, &demo) != nil {
		return nil
	}
	var out []string
	if dsn, ok := dig(demo, "testkube-cloud-api", "api", "postgres", "dsn").(string); ok {
		if u, err := url.Parse(dsn); err == nil {
			if pw, ok := u.User.Password(); ok {
				out = append(out, pw)
			}
		}
	}
	if secret, ok := dig(demo, "testkube-cloud-api", "api", "oauth", "clientSecret").(string); ok {
		out = append(out, secret)
	}
	return out
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

// Best effort: a failed save never hides the install error.
func (i *Installer) saveReport(ctx context.Context, since time.Time, installErr error, ports Ports, secrets ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
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
	rd := newRedactor(append(secrets, demoSecrets()...)...)
	r := buildReport(snap, stuck)
	for _, p := range r.services {
		if !p.ready && p.phase != string(corev1.PodSucceeded) {
			if out, err := i.logs(ctx, p.pod, false, reportLogLines); err == nil && len(out) > 0 {
				// Mask before cutting: a cut secret no longer matches.
				r.logsByPod[p.pod] = cutLines(rd.clean(string(out)))
			}
		}
	}
	now := time.Now()
	text := fmt.Sprintf("Testkube install report, saved after a failed install on %s.\n"+
		"Your license key and Testkube passwords are masked.\n\n%s\nError: %s\n\n%s",
		now.Format("2 Jan 2006 at 15:04"), i.machine(ctx, ports), installErr, renderReport(r, newRedactor()))
	text = capReport(rd.clean(text))
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

// Support's first questions, answered before they ask.
func (i *Installer) machine(ctx context.Context, ports Ports) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CLI: %s\n", i.cliVersion)
	fmt.Fprintf(&b, "Testkube: %s (charts: enterprise %s, runner %s)\n", AppVersion, EnterpriseChartVersion, RunnerChartVersion)
	fmt.Fprintf(&b, "Tools: kind %s, helm %s, node %s\n", ToolVersion("kind"), ToolVersion("helm"), nodeImage)
	fmt.Fprintf(&b, "OS: %s\n", (&Checker{host: realHost{}}).CheckOS().Detail)
	if d := i.docker(ctx); d.Engine != "" {
		fmt.Fprintf(&b, "Docker: %s %s, %d CPUs, %s memory\n", d.Engine, d.Version, d.CPUs, d.MemoryText())
	}
	var mapped []string
	for _, p := range clusterPorts {
		if host, ok := ports[p.name]; ok {
			mapped = append(mapped, fmt.Sprintf("%s %d", p.name, host))
		}
	}
	fmt.Fprintf(&b, "Ports: %s\n", strings.Join(mapped, ", "))
	return b.String()
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

// After masking, so a cut never leaves half a secret.
func capReport(text string) string {
	if len(text) <= maxReportBytes {
		return text
	}
	return cutAt(text, maxReportBytes) + "\n[cut at 1 MB]\n"
}

// Back to a whole character, so the text stays valid.
func cutAt(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func cutLines(out string) []string {
	all := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var lines []string
	for _, line := range all[max(0, len(all)-reportLogLines):] {
		if len(line) > 2048 {
			line = cutAt(line, 2048) + "…"
		}
		lines = append(lines, line)
	}
	return lines
}
