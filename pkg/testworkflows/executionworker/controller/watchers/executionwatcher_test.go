package watchers

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

// TestExecutionWatcher_State checks that the state holds all the events that exist before the watch starts.
// A restarted runner lists them at once, and no later update comes while the pod waits.
func TestExecutionWatcher_State(t *testing.T) {
	const id = "exec-1"
	start := time.Now().Add(-time.Minute)

	tests := []struct {
		name   string
		events int
	}{
		{name: "holds one event", events: 1},
		{name: "holds many events that the list delivers in one update", events: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []runtime.Object{
				&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: "ns", ResourceVersion: "1", CreationTimestamp: metav1.NewTime(start)}},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: id + "-pod", Namespace: "ns", ResourceVersion: "2", Labels: map[string]string{constants.ResourceIdLabelName: id}, CreationTimestamp: metav1.NewTime(start)},
					Status:     corev1.PodStatus{Phase: corev1.PodPending},
				},
			}
			for i := 0; i < tt.events; i++ {
				objects = append(objects, &corev1.Event{
					ObjectMeta:     metav1.ObjectMeta{Name: fmt.Sprintf("event-%d", i), Namespace: "ns", ResourceVersion: fmt.Sprintf("%d", 100+i), CreationTimestamp: metav1.NewTime(start)},
					InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: id + "-pod"},
					Reason:         "FailedMount",
					Type:           corev1.EventTypeWarning,
				})
			}
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			watcher := NewExecutionWatcher(ctx, fake.NewSimpleClientset(objects...), "ns", id, nil, start)

			assert.Eventually(t, func() bool {
				return len(watcher.State().PodEvents().Original()) == tt.events
			}, 2*time.Second, 10*time.Millisecond, "the state does not hold all the pod events")
		})
	}
}

// TestExecutionWatcher_FinalEvents uses a watch that never delivers, as a busy API server can do,
// so only a list can bring the events. The execution ends after the first list of the events.
func TestExecutionWatcher_FinalEvents(t *testing.T) {
	const id = "exec-1"
	start := time.Now().Add(-time.Minute)
	finished := metav1.NewTime(start.Add(30 * time.Second))
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: "ns", ResourceVersion: "1", CreationTimestamp: metav1.NewTime(start)},
		Status:     batchv1.JobStatus{CompletionTime: &finished},
	}
	event := corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: id + ".1", Namespace: "ns", UID: "event-1", ResourceVersion: "2"},
		InvolvedObject: corev1.ObjectReference{Kind: "Job", Name: id},
		Reason:         "DeadlineExceeded",
		Type:           corev1.EventTypeWarning,
		LastTimestamp:  finished,
	}

	tests := []struct {
		name string
		// lists holds the events of each list call in order. The last entry answers every later call.
		lists      [][]corev1.Event
		wantEvents []string
	}{
		{
			name:       "an event that exists only after the first list is in the final state",
			lists:      [][]corev1.Event{nil, {event}},
			wantEvents: []string{"DeadlineExceeded"},
		},
		{
			name:       "an event that the first list returns is in the final state one time",
			lists:      [][]corev1.Event{{event}},
			wantEvents: []string{"DeadlineExceeded"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(job)
			var calls atomic.Int32
			client.PrependReactor("list", "events", func(k8stesting.Action) (bool, runtime.Object, error) {
				i := min(int(calls.Add(1))-1, len(tt.lists)-1)
				return true, &corev1.EventList{ListMeta: metav1.ListMeta{ResourceVersion: "2"}, Items: tt.lists[i]}, nil
			})
			client.PrependWatchReactor("events", func(k8stesting.Action) (bool, watch.Interface, error) {
				return true, watch.NewFake(), nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			watcher := NewExecutionWatcher(ctx, client, "ns", id, nil, start)

			select {
			case <-watcher.Started():
			case <-time.After(5 * time.Second):
				t.Fatal("the watcher did not commit a state")
			}
			if !assert.True(t, watcher.State().Completed()) {
				return
			}
			var reasons []string
			for _, ev := range watcher.State().JobEvents().Original() {
				reasons = append(reasons, ev.Reason)
			}
			assert.Equal(t, tt.wantEvents, reasons)
		})
	}
}
