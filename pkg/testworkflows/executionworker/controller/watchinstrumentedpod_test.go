package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

// fakeWatcher is an execution watcher with a state that the test changes. Each change sends one update.
type fakeWatcher struct {
	mu      sync.Mutex
	state   watchers.ExecutionState
	updates chan struct{}
}

func newFakeWatcher(pod *corev1.Pod) *fakeWatcher {
	w := &fakeWatcher{updates: make(chan struct{}, 10)}
	w.set(pod)
	return w
}

func (w *fakeWatcher) set(pod *corev1.Pod) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.state = watchers.NewExecutionState(nil, watchers.NewPod(pod), watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil)
}

func (w *fakeWatcher) update(pod *corev1.Pod) {
	w.set(pod)
	w.updates <- struct{}{}
}

func (w *fakeWatcher) State() watchers.ExecutionState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}
func (w *fakeWatcher) Commit()                    {}
func (w *fakeWatcher) JobEventsErr() error        { return nil }
func (w *fakeWatcher) PodEventsErr() error        { return nil }
func (w *fakeWatcher) JobErr() error              { return nil }
func (w *fakeWatcher) PodErr() error              { return nil }
func (w *fakeWatcher) RefreshPod(context.Context) {}
func (w *fakeWatcher) RefreshJob(context.Context) {}
func (w *fakeWatcher) Started() <-chan struct{}   { return nil }

// Updated forwards the updates of the test and closes the channel when the context ends, like the real watcher.
func (w *fakeWatcher) Updated(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.updates:
				select {
				case out <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
func (w *fakeWatcher) Next() <-chan struct{} { return nil }

// The watch reads the state after each update, so the test sends the next pod only after it saw the expected message.
func TestWatchInstrumentedPod(t *testing.T) {
	pending := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	watcher := newFakeWatcher(pending)

	ch, err := WatchInstrumentedPod(ctx, nil, nil, time.Time{}, watcher, WatchInstrumentedPodOptions{})
	require.NoError(t, err)

	deadline := time.After(2 * time.Second)
	for _, step := range []struct {
		pod         *corev1.Pod
		wantMessage string
	}{
		{pod: unschedulablePod, wantMessage: "no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu."},
		{pod: pending, wantMessage: ""},
	} {
		watcher.update(step.pod)
		lastMessage := "<no result>"
		for lastMessage != step.wantMessage {
			select {
			case n := <-ch:
				if n.Value.Result != nil {
					lastMessage = n.Value.Result.Initialization.ErrorMessage
				}
			case <-deadline:
				t.Fatalf("the watch did not send the message %q, the last message is %q", step.wantMessage, lastMessage)
			}
		}
	}
}

// staticImageInspector returns the same image data for each image, so the processor needs no registry.
type staticImageInspector struct{}

func (staticImageInspector) Inspect(context.Context, string, string, corev1.PullPolicy, []string) (*imageinspector.Info, error) {
	return &imageinspector.Info{Entrypoint: []string{"/entrypoint"}, User: 1, Group: 1}, nil
}

func (staticImageInspector) ResolveName(_, image string) string { return image }

// bundledPod returns the job, the pod and the signature that the processor makes for the workflow.
// The job and the pod exist since the creation time. The step containers have the state, and for a running state
// the init containers already terminated.
func bundledPod(t *testing.T, workflow *testworkflowsv1.TestWorkflow, created time.Time, state corev1.ContainerState) (*batchv1.Job, *corev1.Pod, []stage.Signature) {
	t.Helper()
	bundle, err := presets.NewPro(staticImageInspector{}).Bundle(context.Background(), workflow, testworkflowprocessor.BundleOptions{
		Config:      testworkflowconfig.InternalConfig{Resource: testworkflowconfig.ResourceConfig{Id: "exec-1", RootId: "exec-1"}},
		ScheduledAt: created,
	})
	require.NoError(t, err)
	job := bundle.Job
	job.CreationTimestamp = metav1.NewTime(created)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "exec-1-pod", Annotations: job.Spec.Template.Annotations, CreationTimestamp: metav1.NewTime(created)},
		Spec:       job.Spec.Template.Spec,
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	initState := state
	if state.Running != nil {
		pod.Status.Phase = corev1.PodRunning
		pod.Status.StartTime = &state.Running.StartedAt
		initState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: state.Running.StartedAt, FinishedAt: state.Running.StartedAt}}
	}
	for _, c := range pod.Spec.InitContainers {
		pod.Status.InitContainerStatuses = append(pod.Status.InitContainerStatuses, corev1.ContainerStatus{Name: c.Name, State: initState})
	}
	for _, c := range pod.Spec.Containers {
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{Name: c.Name, State: state})
	}
	return &job, pod, stage.MapSignatureList(stage.MapSignatureListToInternal(bundle.Signature))
}

// TestWatchInstrumentedPod_InitializationTimeout checks the abort request of the watch for the initialization timeout.
func TestWatchInstrumentedPod_InitializationTimeout(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
		TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Timeouts: &testworkflowsv1.TestWorkflowTimeouts{Initialization: "30s"}},
		Steps:                []testworkflowsv1.Step{{StepOperations: testworkflowsv1.StepOperations{Shell: "sleep 600"}}},
	}}
	// The job is old, so the timer of a watch that connects now fires at once.
	created := time.Now().Add(-5 * time.Minute)
	startedInTime := metav1.NewTime(created.Add(10 * time.Second))

	tests := []struct {
		name      string
		state     corev1.ContainerState
		wantAbort bool
	}{
		{
			name:      "asks for the abort when no step container starts before the timeout ends",
			state:     corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}},
			wantAbort: true,
		},
		{
			name:      "asks for no abort when the watch connects late to a step container that started in time",
			state:     corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: startedInTime}},
			wantAbort: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, pod, signature := bundledPod(t, workflow, created, tt.state)
			watcher := &fakeWatcher{updates: make(chan struct{}, 10)}
			watcher.state = watchers.NewExecutionState(watchers.NewJob(job), watchers.NewPod(pod), watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			ch, err := WatchInstrumentedPod(ctx, fake.NewSimpleClientset(), signature, job.CreationTimestamp.Time, watcher, WatchInstrumentedPodOptions{})
			require.NoError(t, err)

			gotAbort := false
			wait := time.After(time.Second)
		read:
			for !gotAbort {
				select {
				case n, ok := <-ch:
					if !ok {
						break read
					}
					gotAbort = n.Value.AbortReason == testkube.StopReasonInitTimeout
				case <-wait:
					break read
				}
			}
			assert.Equal(t, tt.wantAbort, gotAbort)
		})
	}
}

func TestWaitForUpdate(t *testing.T) {
	tests := []struct {
		name          string
		deadlineFired bool
		expired       bool
		closeUpdates  bool
		wantOk        bool
		wantAborts    int
		wantExpired   bool
	}{
		{
			name:         "returns false when the updates end",
			closeUpdates: true,
			wantOk:       false,
		},
		{
			name:          "returns without an update when the deadline ends, so the caller reads the state one more time",
			deadlineFired: true,
			wantOk:        true,
			wantExpired:   true,
		},
		{
			name:       "asks one time for the abort when the caller waits again after the deadline ended",
			expired:    true,
			wantOk:     true,
			wantAborts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			n := &notifier{ctx: ctx, ch: make(chan ChannelMessage[Notification], 10)}
			updates := make(chan struct{}, 1)
			deadline := initializationDeadline{expired: tt.expired}
			if tt.deadlineFired {
				fired := make(chan time.Time, 1)
				fired <- time.Now()
				deadline.timer = fired
			} else if tt.closeUpdates {
				close(updates)
			} else {
				updates <- struct{}{}
			}

			assert.Equal(t, tt.wantOk, waitForUpdate(updates, &deadline, n))
			assert.Nil(t, deadline.timer, "the deadline fires only one time")
			assert.Equal(t, tt.wantExpired, deadline.expired)
			assert.Len(t, n.ch, tt.wantAborts)
			if tt.wantAborts > 0 {
				assert.Equal(t, testkube.StopReasonInitTimeout, (<-n.ch).Value.AbortReason)
			}
		})
	}
}

func TestContainsLeafStep(t *testing.T) {
	signatureSeq := stage.MapSignatureToSequence(stage.MapSignatureList([]testkube.TestWorkflowSignature{
		{Ref: "rgroup", Children: []testkube.TestWorkflowSignature{{Ref: "rstep1"}}},
	}))

	tests := []struct {
		name string
		refs []string
		want bool
	}{
		{name: "returns false for a container that holds only the setup", refs: []string{"", initconstants.InitStepName}, want: false},
		{name: "returns false for a container that holds only a group", refs: []string{"rgroup"}, want: false},
		{name: "returns true for a container that holds a step", refs: []string{initconstants.InitStepName, "rgroup", "rstep1"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containsLeafStep(tt.refs, signatureSeq))
		})
	}
}

func TestEndInitializationDeadline(t *testing.T) {
	at := time.Now().Add(-time.Minute)
	tests := []struct {
		name       string
		fired      bool
		expired    bool
		startedAt  time.Time
		wantAborts int
	}{
		{name: "asks for the abort when the step container started after the deadline", startedAt: at.Add(time.Second), wantAborts: 1},
		{name: "asks for no abort when the step container started before a deadline that ended while the watch waited", expired: true, startedAt: at.Add(-time.Second), wantAborts: 0},
		{name: "asks for no abort when the step container started before a deadline that fired, for a watch that connects late", fired: true, startedAt: at.Add(-time.Second), wantAborts: 0},
		{name: "asks for no abort when the execution completed and the step container did not start", fired: true, wantAborts: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			n := &notifier{ctx: ctx, ch: make(chan ChannelMessage[Notification], 10)}
			timer := make(chan time.Time, 1)
			if tt.fired {
				timer <- time.Now()
			}
			deadline := initializationDeadline{timer: timer, at: at, expired: tt.expired}
			if tt.expired {
				deadline.timer = nil
			}

			endInitializationDeadline(&deadline, tt.startedAt, n)

			assert.Nil(t, deadline.timer)
			assert.False(t, deadline.expired)
			assert.Len(t, n.ch, tt.wantAborts)
			if tt.wantAborts > 0 {
				assert.Equal(t, testkube.StopReasonInitTimeout, (<-n.ch).Value.AbortReason)
			}
		})
	}
}

func TestNewInitializationDeadline(t *testing.T) {
	annotated := func(created time.Time) *batchv1.Job {
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: created}},
			Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{constants.InitializationTimeoutAnnotation: "1m0s"},
			}}},
		}
	}

	old, recent := time.Now().Add(-2*time.Minute), time.Now()
	tests := []struct {
		name         string
		job          *batchv1.Job
		wantDeadline time.Time
		wantFired    bool
	}{
		{name: "returns no deadline without an initialization timeout", job: &batchv1.Job{}},
		{name: "fires at once when the timeout ended since the job creation", job: annotated(old), wantDeadline: old.Add(time.Minute), wantFired: true},
		{name: "does not fire before the timeout ends", job: annotated(recent), wantDeadline: recent.Add(time.Minute), wantFired: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := watchers.NewExecutionState(watchers.NewJob(tt.job), nil, watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil)

			deadline, stop := newInitializationDeadline(state)
			t.Cleanup(stop)

			assert.True(t, tt.wantDeadline.Equal(deadline.at), "the deadline is %v, want %v", deadline.at, tt.wantDeadline)
			assert.Equal(t, tt.wantDeadline.IsZero(), deadline.timer == nil)
			if deadline.timer == nil {
				return
			}
			select {
			case <-deadline.timer:
				assert.True(t, tt.wantFired, "the deadline fired before the timeout ended")
			case <-time.After(100 * time.Millisecond):
				assert.False(t, tt.wantFired, "the deadline did not fire although the timeout ended")
			}
		})
	}
}
