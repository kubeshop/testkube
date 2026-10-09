package localinstall

import (
	"encoding/json"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Stuck names the service behind a failed install, and why.
type Stuck struct {
	Service string
	Pod     string
	// Short and stable, for tracking.
	Reason   string
	Detail   string
	Restarts int32
	// Last lines of a crashed container, for crashes and failed jobs.
	Logs []string
}

type clusterSnapshot struct {
	pods   []corev1.Pod
	jobs   []batchv1.Job
	events []corev1.Event
}

func parseSnapshot(data []byte) (clusterSnapshot, error) {
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return clusterSnapshot{}, err
	}
	var s clusterSnapshot
	for _, item := range list.Items {
		var meta metav1.TypeMeta
		if err := json.Unmarshal(item, &meta); err != nil {
			return clusterSnapshot{}, err
		}
		var err error
		switch meta.Kind {
		case "Pod":
			s.pods = append(s.pods, corev1.Pod{})
			err = json.Unmarshal(item, &s.pods[len(s.pods)-1])
		case "Job":
			s.jobs = append(s.jobs, batchv1.Job{})
			err = json.Unmarshal(item, &s.jobs[len(s.jobs)-1])
		case "Event":
			s.events = append(s.events, corev1.Event{})
			err = json.Unmarshal(item, &s.events[len(s.events)-1])
		}
		if err != nil {
			return clusterSnapshot{}, err
		}
	}
	return s, nil
}

type stuckService struct {
	name, label string
	// Role first for users, name kept for support.
	title string
	job   bool
	deps  []string
}

// Root first, so a dependency is blamed before its users.
var stuckServices = []stuckService{
	{name: "postgres", label: "postgresql", title: "The database (postgres)"},
	{name: "nats", label: "nats", title: "Messaging (nats)"},
	{name: "minio", label: "minio", title: "File storage (minio)"},
	{name: "dex", label: "dex", title: "Login (dex)"},
	{name: "ui", label: "testkube-cloud-ui", title: "The dashboard (ui)"},
	{name: "migration", label: "testkube-enterprise-api-migration", title: "Database setup (migration)", job: true,
		deps: []string{"postgres"}},
	{name: "api", label: "testkube-cloud-api", title: "The API (api)", deps: []string{"postgres", "nats", "minio", "migration"}},
	{name: "worker-service", label: "testkube-worker-service", title: "Background jobs (worker-service)", deps: []string{"nats", "minio"}},
	{name: "ai-service", label: "testkube-ai-service", title: "The AI assistant (ai-service)", deps: []string{"postgres"}},
	{name: "runner", label: "testkube-runner", title: "The runner", deps: []string{"api"}},
}

func (s Stuck) Title() string {
	for _, svc := range stuckServices {
		if svc.name == s.Service {
			return svc.title
		}
	}
	return s.Service
}

// Covers kubelet's 300s restart backoff after dependencies come up.
const depsSettle = 6 * time.Minute

var imageMissingWords = []string{"not found", "manifest unknown", "denied", "unauthorized", "does not exist"}

// Blames the root cause, never a service waiting on it.
func findStuck(s clusterSnapshot, since, now time.Time) (Stuck, bool) {
	pods := map[string]*corev1.Pod{}
	for i := range s.pods {
		if name := serviceOf(&s.pods[i]); name != "" && pods[name] == nil {
			pods[name] = &s.pods[i]
		}
	}
	// Waiting can't fix these, wherever they are.
	for _, svc := range stuckServices {
		if p := pods[svc.name]; p != nil {
			if st, ok := hopeless(svc.name, p, s, since); ok {
				return st, true
			}
		}
	}
	readySince := map[string]time.Time{}
	for _, svc := range stuckServices {
		if t, ok := serviceReadySince(svc, pods[svc.name], s.jobs); ok {
			readySince[svc.name] = t
		}
	}
	settled := func(svc stuckService) bool {
		for _, d := range svc.deps {
			if t, ok := readySince[d]; !ok || now.Sub(t) < depsSettle {
				return false
			}
		}
		return true
	}
	for _, svc := range stuckServices {
		if _, ok := readySince[svc.name]; ok {
			continue
		}
		if svc.job {
			if job := findJob(s.jobs, svc.label); job != nil && jobFailed(job) && settled(svc) {
				return Stuck{Service: svc.name, Reason: "job_failed", Detail: job.Name, Pod: podName(pods[svc.name])}, true
			}
			continue
		}
		p := pods[svc.name]
		if p == nil {
			continue
		}
		if st, ok := pulling(svc.name, p, s.events); ok {
			return st, true
		}
		threshold := int32(2)
		if len(svc.deps) > 0 {
			threshold = 3
		}
		if n := restarts(p, since); n >= threshold && settled(svc) {
			return Stuck{Service: svc.name, Pod: p.Name, Reason: "crashloop", Restarts: n}, true
		}
	}
	// Nothing explains it, so the first unready one is root.
	for _, svc := range stuckServices {
		if _, ok := readySince[svc.name]; ok || pods[svc.name] == nil {
			continue
		}
		p := pods[svc.name]
		detail := "waiting to start"
		if p.Status.Phase == corev1.PodRunning {
			detail = "running but never ready"
		}
		return Stuck{Service: svc.name, Pod: p.Name, Reason: "not_ready", Detail: detail, Restarts: restarts(p, since)}, true
	}
	return Stuck{}, false
}

func serviceOf(p *corev1.Pod) string {
	for _, svc := range stuckServices {
		if svc.job && strings.HasPrefix(p.Labels["job-name"], svc.label) ||
			!svc.job && p.Labels["app.kubernetes.io/name"] == svc.label {
			return svc.name
		}
	}
	return ""
}

func hopeless(name string, p *corev1.Pod, s clusterSnapshot, since time.Time) (Stuck, bool) {
	st := Stuck{Service: name, Pod: p.Name}
	for _, c := range p.Status.ContainerStatuses {
		if t := c.LastTerminationState.Terminated; t != nil && t.Reason == "OOMKilled" && t.FinishedAt.After(since) {
			st.Reason, st.Detail, st.Restarts = "oom", memoryLimit(p, c.Name), c.RestartCount
			return st, true
		}
		if w := c.State.Waiting; w != nil {
			switch w.Reason {
			case "CreateContainerConfigError":
				st.Reason, st.Detail = "config", w.Message
				return st, true
			case "ErrImagePull", "ImagePullBackOff":
				msg := latestEvent(s.events, "Pod", p.Name, "Failed")
				switch {
				case strings.Contains(msg, "toomanyrequests"):
					st.Reason, st.Detail = "rate_limit", c.Image
					return st, true
				case containsAny(msg, imageMissingWords):
					st.Reason, st.Detail = "image_missing", c.Image
					return st, true
				}
			}
		}
	}
	for _, cond := range p.Status.Conditions {
		if cond.Type != corev1.PodScheduled || cond.Status != corev1.ConditionFalse || cond.Reason != "Unschedulable" {
			continue
		}
		switch {
		case strings.Contains(cond.Message, "Insufficient cpu"):
			st.Reason = "no_cpu"
		case strings.Contains(cond.Message, "Insufficient memory"):
			st.Reason = "no_memory"
		case strings.Contains(cond.Message, "unbound"):
			st.Reason, st.Detail = "storage", storageReason(p, s)
		default:
			st.Reason, st.Detail = "unschedulable", cond.Message
		}
		return st, true
	}
	return Stuck{}, false
}

// Slow downloads and network errors may still pass.
func pulling(name string, p *corev1.Pod, events []corev1.Event) (Stuck, bool) {
	for _, c := range p.Status.ContainerStatuses {
		w := c.State.Waiting
		if w == nil {
			continue
		}
		switch {
		case w.Reason == "ErrImagePull" || w.Reason == "ImagePullBackOff":
			return Stuck{Service: name, Pod: p.Name, Reason: "image_pull", Detail: c.Image}, true
		case w.Reason == "ContainerCreating" && latestEvent(events, "Pod", p.Name, "Pulling") != "" &&
			latestEvent(events, "Pod", p.Name, "Pulled") == "":
			return Stuck{Service: name, Pod: p.Name, Reason: "downloading", Detail: c.Image}, true
		}
	}
	return Stuck{}, false
}

// Docker restarts show as Unknown; they aren't the pod's fault.
func restarts(p *corev1.Pod, since time.Time) int32 {
	var n int32
	for _, c := range p.Status.ContainerStatuses {
		if t := c.LastTerminationState.Terminated; t != nil && t.Reason != "Unknown" && t.FinishedAt.After(since) {
			n = max(n, c.RestartCount)
		}
	}
	return n
}

func serviceReadySince(svc stuckService, p *corev1.Pod, jobs []batchv1.Job) (time.Time, bool) {
	if svc.job {
		if job := findJob(jobs, svc.label); job != nil {
			for _, c := range job.Status.Conditions {
				if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
					return c.LastTransitionTime.Time, true
				}
			}
		}
		return time.Time{}, false
	}
	if p == nil {
		return time.Time{}, false
	}
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return c.LastTransitionTime.Time, true
		}
	}
	return time.Time{}, false
}

// The newest job wins: a retry replaces the failed one.
func findJob(jobs []batchv1.Job, prefix string) *batchv1.Job {
	var found *batchv1.Job
	for i := range jobs {
		if strings.HasPrefix(jobs[i].Name, prefix) &&
			(found == nil || jobs[i].CreationTimestamp.After(found.CreationTimestamp.Time)) {
			found = &jobs[i]
		}
	}
	return found
}

func jobFailed(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func latestEvent(events []corev1.Event, kind, name, reason string) string {
	var msg string
	var at time.Time
	for _, e := range events {
		if e.InvolvedObject.Kind != kind || e.InvolvedObject.Name != name || e.Reason != reason {
			continue
		}
		t := e.LastTimestamp.Time
		if t.IsZero() {
			t = e.EventTime.Time
		}
		if msg == "" || t.After(at) {
			msg, at = e.Message, t
		}
	}
	return msg
}

// Only the volume's event says why; its status doesn't.
func storageReason(p *corev1.Pod, s clusterSnapshot) string {
	for _, v := range p.Spec.Volumes {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		if msg := latestEvent(s.events, "PersistentVolumeClaim", v.PersistentVolumeClaim.ClaimName, "ProvisioningFailed"); msg != "" {
			return msg
		}
	}
	return "storage not ready"
}

func memoryLimit(p *corev1.Pod, container string) string {
	for _, c := range p.Spec.Containers {
		if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok && c.Name == container {
			return q.String()
		}
	}
	return ""
}

func podName(p *corev1.Pod) string {
	if p == nil {
		return ""
	}
	return p.Name
}

// Our services log JSON; people read the message.
func readableLogLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		var entry struct {
			Msg   string `json:"msg"`
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(line), &entry) == nil && entry.Msg != "" {
			line = entry.Msg
			if entry.Error != "" {
				line += ": " + entry.Error
			}
		}
		if line == "" {
			continue
		}
		if r := []rune(line); len(r) > 200 {
			line = string(r[:200]) + "…"
		}
		lines = append(lines, line)
	}
	return lines
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
