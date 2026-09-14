package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
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
