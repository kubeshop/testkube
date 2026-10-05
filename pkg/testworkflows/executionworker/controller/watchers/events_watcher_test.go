package watchers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

// fakeEventsClient lists the events that the test sets, and watches through a fake watcher.
type fakeEventsClient struct {
	mu      sync.Mutex
	list    corev1.EventList
	watcher *watch.FakeWatcher
}

func (c *fakeEventsClient) setList(list corev1.EventList) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.list = list
}

func (c *fakeEventsClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.EventList, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	list := c.list
	return &list, nil
}

func (c *fakeEventsClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return c.watcher, nil
}

// eventRecorder keeps the events that the watcher sends to its listener.
type eventRecorder struct {
	mu     sync.Mutex
	events []string
}

func (r *eventRecorder) listen(event *corev1.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event.Reason+"@"+event.ResourceVersion)
}

func (r *eventRecorder) received() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

func waitingEvent(uid, reason, resourceVersion string) corev1.Event {
	return corev1.Event{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid), Name: uid, Namespace: "ns", ResourceVersion: resourceVersion},
		Reason:     reason,
	}
}

func TestEventsWatcher_List(t *testing.T) {
	tests := []struct {
		name  string
		list  corev1.EventList
		watch []corev1.Event
		// relist is the list that an update reads after the watch delivered its events.
		relist *corev1.EventList
		want   []string
	}{
		{
			name: "the newest event has the version of the list and still reaches the listener",
			list: corev1.EventList{
				ListMeta: metav1.ListMeta{ResourceVersion: "12"},
				Items:    []corev1.Event{waitingEvent("a", "Scheduled", "10"), waitingEvent("b", "FailedMount", "12")},
			},
			want: []string{"Scheduled@10", "FailedMount@12"},
		},
		{
			name: "events older than the list reach the listener",
			list: corev1.EventList{
				ListMeta: metav1.ListMeta{ResourceVersion: "20"},
				Items:    []corev1.Event{waitingEvent("a", "Scheduled", "10"), waitingEvent("b", "FailedMount", "12")},
			},
			want: []string{"Scheduled@10", "FailedMount@12"},
		},
		{
			name: "a list after the watch does not send an event version again",
			list: corev1.EventList{
				ListMeta: metav1.ListMeta{ResourceVersion: "12"},
				Items:    []corev1.Event{waitingEvent("a", "Scheduled", "10")},
			},
			watch: []corev1.Event{waitingEvent("b", "FailedMount", "13")},
			relist: &corev1.EventList{
				ListMeta: metav1.ListMeta{ResourceVersion: "14"},
				Items:    []corev1.Event{waitingEvent("a", "Scheduled", "10"), waitingEvent("b", "FailedMount", "13"), waitingEvent("b", "FailedMount", "14")},
			},
			want: []string{"Scheduled@10", "FailedMount@13", "FailedMount@14"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			client := &fakeEventsClient{list: tt.list, watcher: watch.NewFakeWithChanSize(len(tt.watch), false)}
			recorder := &eventRecorder{}

			watcher := NewEventsWatcher(ctx, client, metav1.ListOptions{}, recorder.listen)
			<-watcher.Started()
			for i := range tt.watch {
				client.watcher.Modify(&tt.watch[i])
			}
			delivered := len(tt.list.Items) + len(tt.watch)
			require.Eventually(t, func() bool { return len(recorder.received()) >= delivered }, 2*time.Second, 10*time.Millisecond)

			if tt.relist != nil {
				client.setList(*tt.relist)
				_, err := watcher.Update(0)
				require.NoError(t, err)
				require.Eventually(t, func() bool { return len(recorder.received()) >= len(tt.want) }, 2*time.Second, 10*time.Millisecond)
			}

			assert.Equal(t, tt.want, recorder.received())
		})
	}
}
