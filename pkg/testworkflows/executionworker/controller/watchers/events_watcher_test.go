package watchers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

// fakeEventsClient lists the events that the test sets, and watches through a fake watcher. The
// list and the watch return the queued errors first, one for each call. A list that succeeds
// takes the next queued list, and keeps the last one.
type fakeEventsClient struct {
	mu        sync.Mutex
	list      corev1.EventList
	next      []corev1.EventList
	watcher   *watch.FakeWatcher
	listErrs  []error
	watchErrs []error
}

func (c *fakeEventsClient) setList(list corev1.EventList) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.list = list
}

func (c *fakeEventsClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.EventList, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.listErrs) > 0 {
		err := c.listErrs[0]
		c.listErrs = c.listErrs[1:]
		return nil, err
	}
	if len(c.next) > 0 {
		c.list, c.next = c.next[0], c.next[1:]
	}
	list := c.list
	return &list, nil
}

func (c *fakeEventsClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.watchErrs) > 0 {
		err := c.watchErrs[0]
		c.watchErrs = c.watchErrs[1:]
		return nil, err
	}
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

			watcher := NewEventsWatcher(ctx, client, metav1.ListOptions{}, nil, recorder.listen)
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

func TestEventsWatcher_Retry(t *testing.T) {
	delay, maxDelay := eventsRetryDelay, eventsRetryMaxDelay
	t.Cleanup(func() { eventsRetryDelay, eventsRetryMaxDelay = delay, maxDelay })
	eventsRetryDelay, eventsRetryMaxDelay = time.Millisecond, 5*time.Millisecond

	busy := apierrors.NewTooManyRequests("the server is busy", 1)
	list := corev1.EventList{
		ListMeta: metav1.ListMeta{ResourceVersion: "12"},
		Items:    []corev1.Event{waitingEvent("a", "Scheduled", "10"), waitingEvent("b", "FailedMount", "12")},
	}
	// later is the list after one more event, which only a second list can read.
	later := corev1.EventList{
		ListMeta: metav1.ListMeta{ResourceVersion: "13"},
		Items:    append(append([]corev1.Event{}, list.Items...), waitingEvent("b", "FailedMount", "13")),
	}
	tests := []struct {
		name      string
		listErrs  []error
		watchErrs []error
		// expired sends an error for a version that is too old into the first watch.
		expired bool
		want    []string
	}{
		{
			name:     "a list that fails gets the events on the next list",
			listErrs: []error{busy, busy},
			want:     []string{"Scheduled@10", "FailedMount@12"},
		},
		{
			name:      "a watch that fails gets the new events from the next list",
			watchErrs: []error{busy},
			want:      []string{"Scheduled@10", "FailedMount@12", "FailedMount@13"},
		},
		{
			name:    "a version that is too old gets the new events from the next list",
			expired: true,
			want:    []string{"Scheduled@10", "FailedMount@12", "FailedMount@13"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			client := &fakeEventsClient{
				next:      []corev1.EventList{list, later},
				listErrs:  tt.listErrs,
				watchErrs: tt.watchErrs,
				watcher:   watch.NewFakeWithChanSize(1, false),
			}
			if tt.expired {
				client.watcher.Error(&apierrors.NewResourceExpired("too old resource version").ErrStatus)
			}
			recorder := &eventRecorder{}

			watcher := NewEventsWatcher(ctx, client, metav1.ListOptions{}, nil, recorder.listen)
			require.Eventually(t, func() bool { return len(recorder.received()) >= len(tt.want) }, 2*time.Second, time.Millisecond)

			assert.Equal(t, tt.want, recorder.received())
			assert.NoError(t, watcher.Err())
		})
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		name    string
		watched []bool
		want    []time.Duration
	}{
		{
			name:    "errors in a row double the delay up to the maximum",
			watched: []bool{false, false, false, false},
			want:    []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second},
		},
		{
			name:    "a watch that worked starts the delay again from the base",
			watched: []bool{false, false, true, false},
			want:    []time.Duration{time.Second, 2 * time.Second, time.Second, 2 * time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retry := backoff{base: time.Second, max: 4 * time.Second}
			got := make([]time.Duration, 0, len(tt.watched))
			for _, watched := range tt.watched {
				got = append(got, retry.delay(watched))
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEventsWatcher_ListAndWatch(t *testing.T) {
	busy := apierrors.NewTooManyRequests("the server is busy", 1)
	list := corev1.EventList{
		ListMeta: metav1.ListMeta{ResourceVersion: "12"},
		Items:    []corev1.Event{waitingEvent("a", "Scheduled", "10")},
	}
	tests := []struct {
		name      string
		listErrs  []error
		watchErrs []error
		// delivered is the number of events that the watch sends before its error.
		delivered   int
		wantWatched bool
	}{
		{
			name:     "a list that fails is not a watch that worked",
			listErrs: []error{busy},
		},
		{
			name:      "a list that works and a watch that the API refuses is not a watch that worked",
			watchErrs: []error{busy},
		},
		{
			name:        "a watch that delivers an event before its error worked",
			delivered:   1,
			wantWatched: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			t.Cleanup(func() { cancel(nil) })
			client := &fakeEventsClient{
				list:      list,
				listErrs:  tt.listErrs,
				watchErrs: tt.watchErrs,
				watcher:   watch.NewFakeWithChanSize(tt.delivered+1, false),
			}
			for i := 0; i < tt.delivered; i++ {
				event := waitingEvent("b", "FailedMount", "13")
				client.watcher.Add(&event)
			}
			client.watcher.Error(&busy.ErrStatus)
			watcher := &eventsWatcher{
				client:    client,
				listener:  func(*corev1.Event) {},
				optsCh:    make(chan struct{}),
				startedCh: make(chan struct{}),
				ctx:       ctx,
				cancel:    cancel,
				sent:      make(map[string]struct{}),
			}

			watched, err := watcher.listAndWatch()

			assert.Equal(t, tt.wantWatched, watched)
			assert.True(t, apierrors.IsTooManyRequests(err), "the error of the list or the watch: %v", err)
		})
	}
}

// TestEventsWatcher_RetryUntil checks that the watcher stops after an error once no other watcher
// observes the execution, because the caller waits for all watchers to end before it reports.
func TestEventsWatcher_RetryUntil(t *testing.T) {
	delay, maxDelay := eventsRetryDelay, eventsRetryMaxDelay
	t.Cleanup(func() { eventsRetryDelay, eventsRetryMaxDelay = delay, maxDelay })

	busy := apierrors.NewTooManyRequests("the server is busy", 1)
	list := corev1.EventList{
		ListMeta: metav1.ListMeta{ResourceVersion: "12"},
		Items:    []corev1.Event{waitingEvent("a", "Scheduled", "10")},
	}
	tests := []struct {
		name string
		// observed tells whether other watchers still observe the execution when it starts.
		observed bool
		// endObservation ends the other watchers while the watcher waits to retry.
		endObservation bool
		retryDelay     time.Duration
		wantDone       bool
	}{
		{
			name:       "an error when no other watcher observes the execution ends the watcher",
			retryDelay: time.Millisecond,
			wantDone:   true,
		},
		{
			name:           "a watcher that waits to retry ends when the other watchers end",
			observed:       true,
			endObservation: true,
			retryDelay:     time.Hour,
			wantDone:       true,
		},
		{
			name:       "an error while other watchers observe the execution is retried",
			observed:   true,
			retryDelay: time.Millisecond,
			wantDone:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventsRetryDelay, eventsRetryMaxDelay = tt.retryDelay, tt.retryDelay
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			client := &fakeEventsClient{list: list, listErrs: []error{busy}, watcher: watch.NewFakeWithChanSize(1, false)}
			observation := make(chan struct{})
			if !tt.observed {
				close(observation)
			}
			recorder := &eventRecorder{}

			watcher := NewEventsWatcher(ctx, client, metav1.ListOptions{}, observation, recorder.listen)
			<-watcher.Started()
			if tt.endObservation {
				close(observation)
			}

			if tt.wantDone {
				require.Eventually(t, func() bool {
					select {
					case <-watcher.Done():
						return true
					default:
						return false
					}
				}, 2*time.Second, time.Millisecond)
				assert.Error(t, watcher.Err())
				return
			}
			require.Eventually(t, func() bool { return len(recorder.received()) == 1 }, 2*time.Second, time.Millisecond)
			assert.NoError(t, watcher.Err())
		})
	}
}

// slowEventsClient blocks the first list until the context ends, or every list when hangAll is set,
// so a test can hold one read while another read runs.
type slowEventsClient struct {
	calls   atomic.Int32
	hangAll bool
	list    corev1.EventList
	watcher *watch.FakeWatcher
}

func (c *slowEventsClient) List(ctx context.Context, _ metav1.ListOptions) (*corev1.EventList, error) {
	if c.calls.Add(1) == 1 || c.hangAll {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	list := c.list
	return &list, nil
}

func (c *slowEventsClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return c.watcher, nil
}

func TestEventsWatcher_Update(t *testing.T) {
	const limit = 200 * time.Millisecond
	tests := []struct {
		name      string
		hangAll   bool
		wantCount int
		wantErr   bool
	}{
		{
			name:      "a slow list of the watch cycle does not hold back the update",
			wantCount: 1,
		},
		{
			name:    "an update whose own list hangs returns at its limit",
			hangAll: true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			client := &slowEventsClient{
				hangAll: tt.hangAll,
				list:    corev1.EventList{Items: []corev1.Event{waitingEvent("a", "FailedMount", "1")}},
				watcher: watch.NewFake(),
			}
			recorder := &eventRecorder{}
			watcher := NewEventsWatcher(ctx, client, metav1.ListOptions{}, nil, recorder.listen)
			require.Eventually(t, func() bool { return client.calls.Load() >= 1 }, time.Second, time.Millisecond)

			startedAt := time.Now()
			count, err := watcher.Update(limit)

			assert.Less(t, time.Since(startedAt), limit+500*time.Millisecond)
			assert.Equal(t, tt.wantCount, count)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
