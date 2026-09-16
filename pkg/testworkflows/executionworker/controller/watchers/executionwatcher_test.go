package watchers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

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

func TestExecutionWatcher_RefreshPodEvents(t *testing.T) {
	const id = "exec-1"
	start := time.Now().Add(-time.Minute)
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: "ns", ResourceVersion: "1", CreationTimestamp: metav1.NewTime(start)}}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	watcher := NewExecutionWatcher(ctx, fake.NewSimpleClientset(job), "ns", id, nil, start)

	done := make(chan struct{})
	go func() {
		watcher.RefreshPodEvents(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the refresh waits for the events of a pod that does not exist")
	}
}
