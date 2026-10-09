package localinstall

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	installStart = time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	installEnd   = installStart.Add(15 * time.Minute)
)

type podChange func(*corev1.Pod)

func testPod(service string, changes ...podChange) corev1.Pod {
	p := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: service + "-pod", Labels: map[string]string{}}}
	for _, svc := range stuckServices {
		if svc.name == service {
			p.Labels["app.kubernetes.io/name"] = svc.label
		}
	}
	p.Status.Phase = corev1.PodRunning
	p.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "main", Image: "docker.io/kubeshop/" + service + ":1"}}
	for _, change := range changes {
		change(&p)
	}
	return p
}

func readyAt(t time.Time) podChange {
	return func(p *corev1.Pod) {
		p.Status.Conditions = append(p.Status.Conditions, corev1.PodCondition{Type: corev1.PodReady,
			Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(t)})
	}
}

func waitingFor(reason string) podChange {
	return func(p *corev1.Pod) {
		p.Status.Phase = corev1.PodPending
		p.Status.ContainerStatuses[0].State.Waiting = &corev1.ContainerStateWaiting{Reason: reason}
	}
}

func crashed(times int32, reason string, at time.Time) podChange {
	return func(p *corev1.Pod) {
		p.Status.ContainerStatuses[0].RestartCount = times
		p.Status.ContainerStatuses[0].LastTerminationState.Terminated = &corev1.ContainerStateTerminated{
			Reason: reason, ExitCode: 1, FinishedAt: metav1.NewTime(at)}
	}
}

func unschedulable(msg string) podChange {
	return func(p *corev1.Pod) {
		p.Status.Phase = corev1.PodPending
		p.Status.ContainerStatuses = nil
		p.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionFalse,
			Reason: "Unschedulable", Message: msg}}
	}
}

func testEvent(kind, name, reason, msg string) corev1.Event {
	return corev1.Event{InvolvedObject: corev1.ObjectReference{Kind: kind, Name: name}, Reason: reason, Message: msg,
		LastTimestamp: metav1.NewTime(installStart.Add(time.Minute))}
}

func migrationJob(condition batchv1.JobConditionType, at time.Time) batchv1.Job {
	return batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "testkube-enterprise-api-migration-1"},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: condition, Status: corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(at)}}}}
}

// Everything up a minute in; each case breaks one thing.
func healthy(replace ...corev1.Pod) clusterSnapshot {
	up := readyAt(installStart.Add(time.Minute))
	pods := map[string]corev1.Pod{}
	for _, svc := range stuckServices {
		if !svc.job {
			pods[svc.name] = testPod(svc.name, up)
		}
	}
	for _, p := range replace {
		pods[serviceOf(&p)] = p
	}
	s := clusterSnapshot{jobs: []batchv1.Job{migrationJob(batchv1.JobComplete, installStart.Add(2*time.Minute))}}
	for _, svc := range stuckServices {
		if p, ok := pods[svc.name]; ok {
			s.pods = append(s.pods, p)
		}
	}
	return s
}

func TestFindStuck_BlamesTheRootNotTheServicesWaitingOnIt(t *testing.T) {
	apiCrashing := testPod("api", crashed(6, "Error", installEnd.Add(-time.Minute)))
	oomAPI := testPod("api", crashed(2, "OOMKilled", installEnd.Add(-time.Minute)), func(p *corev1.Pod) {
		p.Spec.Containers = []corev1.Container{{Name: "main", Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")}}}}
	})
	pvcPostgres := testPod("postgres", unschedulable("0/1 nodes are available: pod has unbound immediate PersistentVolumeClaims"),
		func(p *corev1.Pod) {
			p.Spec.Volumes = []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data-postgres"}}}}
		})

	tests := map[string]struct {
		snapshot clusterSnapshot
		want     Stuck
		found    bool
	}{
		"all ready": {snapshot: healthy()},
		"missing postgres image, not the crashing api": {
			snapshot: withEvents(healthy(testPod("postgres", waitingFor("ImagePullBackOff")), apiCrashing),
				testEvent("Pod", "postgres-pod", "Failed", "pull access denied, repository does not exist")),
			want: Stuck{Service: "postgres", Pod: "postgres-pod", Reason: "image_missing", Detail: "docker.io/kubeshop/postgres:1"}, found: true,
		},
		"docker hub limit behind kubelet's newer bare event": {
			snapshot: withEvents(healthy(testPod("nats", waitingFor("ImagePullBackOff"))),
				testEvent("Pod", "nats-pod", "Failed", "429 Too Many Requests - Server message: toomanyrequests: You have reached"),
				later(testEvent("Pod", "nats-pod", "Failed", "Error: ImagePullBackOff"))),
			want: Stuck{Service: "nats", Pod: "nats-pod", Reason: "rate_limit", Detail: "docker.io/kubeshop/nats:1"}, found: true,
		},
		"missing image named only in the waiting message": {
			snapshot: healthy(testPod("dex", waitingFor("ErrImagePull"), func(p *corev1.Pod) {
				p.Status.ContainerStatuses[0].State.Waiting.Message = `failed to resolve reference "docker.io/kubeshop/dex:1": not found`
			})),
			want: Stuck{Service: "dex", Pod: "dex-pod", Reason: "image_missing", Detail: "docker.io/kubeshop/dex:1"}, found: true,
		},
		"api out of memory beats a slow postgres download": {
			snapshot: withEvents(healthy(testPod("postgres", waitingFor("ContainerCreating")), oomAPI),
				testEvent("Pod", "postgres-pod", "Pulling", "Pulling image")),
			want: Stuck{Service: "api", Pod: "api-pod", Reason: "oom", Detail: "512Mi", Restarts: 2}, found: true,
		},
		"not enough memory to start": {
			snapshot: healthy(testPod("minio", unschedulable("0/1 nodes are available: 1 Insufficient memory."))),
			want:     Stuck{Service: "minio", Pod: "minio-pod", Reason: "no_memory"}, found: true,
		},
		"storage names the volume's own reason": {
			snapshot: withEvents(healthy(pvcPostgres),
				testEvent("PersistentVolumeClaim", "data-postgres", "ProvisioningFailed", `storageclass "standard" not found`)),
			want: Stuck{Service: "postgres", Pod: "postgres-pod", Reason: "storage", Detail: `storageclass "standard" not found`}, found: true,
		},
		"postgres still downloading, not the crashing api": {
			snapshot: withEvents(healthy(testPod("postgres", waitingFor("ContainerCreating")), apiCrashing),
				testEvent("Pod", "postgres-pod", "Pulling", "Pulling image")),
			want: Stuck{Service: "postgres", Pod: "postgres-pod", Reason: "downloading", Detail: "docker.io/kubeshop/postgres:1"}, found: true,
		},
		"api crashes long after its dependencies came up": {
			snapshot: healthy(apiCrashing),
			want:     Stuck{Service: "api", Pod: "api-pod", Reason: "crashloop", Restarts: 6}, found: true,
		},
		"api crash right after dependencies came up is only slow": {
			snapshot: healthy(apiCrashing, testPod("nats", readyAt(installEnd.Add(-2*time.Minute)))),
			want:     Stuck{Service: "api", Pod: "api-pod", Reason: "not_ready", Detail: "running but never ready", Restarts: 6}, found: true,
		},
		"a service without dependencies crashing twice": {
			snapshot: healthy(testPod("dex", crashed(2, "Error", installEnd.Add(-time.Minute)))),
			want:     Stuck{Service: "dex", Pod: "dex-pod", Reason: "crashloop", Restarts: 2}, found: true,
		},
		"restarts before this install and Docker restarts don't count": {
			snapshot: healthy(testPod("dex", crashed(9, "Error", installStart.Add(-time.Hour))),
				testPod("ui", crashed(3, "Unknown", installEnd.Add(-time.Minute)))),
			want: Stuck{Service: "dex", Pod: "dex-pod", Reason: "not_ready", Detail: "running but never ready"}, found: true,
		},
		"migration failed with postgres up": {
			snapshot: func() clusterSnapshot {
				s := healthy()
				s.jobs = []batchv1.Job{migrationJob(batchv1.JobFailed, installStart.Add(10*time.Minute))}
				return s
			}(),
			want: Stuck{Service: "migration", Reason: "job_failed", Detail: "testkube-enterprise-api-migration-1"}, found: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, found := findStuck(tt.snapshot, installStart, installEnd)

			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.want, got)
		})
	}
}

func later(e corev1.Event) corev1.Event {
	e.LastTimestamp = metav1.NewTime(e.LastTimestamp.Add(time.Minute))
	return e
}

func withEvents(s clusterSnapshot, events ...corev1.Event) clusterSnapshot {
	s.events = events
	return s
}
