package watchers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/internal/common"
)

type jobWatcher struct {
	client  kubernetesClient[batchv1.JobList, *batchv1.Job]
	opts    metav1.ListOptions
	peek    *batchv1.Job
	started atomic.Bool
	ch      chan *batchv1.Job
	peekCh  chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	err     error
	mu      sync.Mutex
	peekMu  sync.Mutex
}

type JobWatcher interface {
	Channel() <-chan *batchv1.Job
	Peek(ctx context.Context) <-chan *batchv1.Job
	Update(t time.Duration) (int, error)
	Started() bool
	Stop()
	Done() <-chan struct{}
	Err() error
}

func NewJobWatcher(parentCtx context.Context, client kubernetesClient[batchv1.JobList, *batchv1.Job], opts metav1.ListOptions, bufferSize int) JobWatcher {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	opts.AllowWatchBookmarks = true
	watcher := &jobWatcher{
		client: client,
		opts:   opts,
		ch:     make(chan *batchv1.Job, bufferSize),
		ctx:    ctx,
		cancel: ctxCancel,
	}
	go watcher.cycle()
	return watcher
}

func (e *jobWatcher) Started() bool {
	return e.started.Load(true)
}

func (e *jobWatcher) setError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.err = err
	e.cancel()
}

func (e *jobWatcher) finalize(job *batchv1.Job) bool {
	if IsJobFinished(job) {
		e.err = ErrDone
		e.cancel()
		return true
	}
	return false
}

func (e *jobWatcher) setLastJob(job *batchv1.Job) {
	e.peekMu.Lock()
	defer e.peekMu.Unlock()
	e.peek = job
	peekCh := e.peekCh
	e.peekCh = nil
	if peekCh == nil {
		close(peekCh)
	}
}

func (e *jobWatcher) read(t time.Duration) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Fetch the data
	opts := e.opts
	opts.ResourceVersion = ""
	if t != 0 {
		opts.TimeoutSeconds = common.Ptr(int64(math.Ceil(t.Seconds())))
	}
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultListTimeoutSeconds)
	}
	list, err := e.client.List(e.ctx, e.opts)
	if err != nil {
		return 0, err
	}

	// Update the latest resource version
	e.opts.ResourceVersion = list.ResourceVersion

	// Ignore error when the channel is already closed
	defer func() {
		recover()
	}()

	// Disallow watching multiple jobs in that watcher
	if len(list.Items) > 1 {
		names := make([]string, len(list.Items))
		for i := range list.Items {
			names[i] = list.Items[i].Name
		}
		return 0, fmt.Errorf("found more than one job for selected criteria: %s", strings.Join(names, ", "))
	}

	// Handle lack of the job
	if len(list.Items) == 0 {
		e.peekMu.Lock()
		job := e.peek
		e.peekMu.Unlock()
		if job == nil {
			// there is no job, but it's not a change.
			return 0, nil
		} else {
			// the job was there, but it's deleted now.
			e.finalize(nil)
			return 1, ErrDone
		}
	}

	// There is no update
	if list.Items[0].ResourceVersion == e.opts.ResourceVersion {
		return 0, nil
	}

	// The job has been updated
	e.ch <- common.Ptr(list.Items[0])
	e.setLastJob(common.Ptr(list.Items[0]))

	return 1, nil
}

func (e *jobWatcher) watch() error {
	// Initialize the watcher
	opts := e.opts
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultWatchTimeoutSeconds)
	}
	watcher, err := e.client.Watch(e.ctx, opts)
	defer watcher.Stop()
	if err != nil {
		return err
	}

	// Ignore error when the channel is already closed
	defer func() {
		recover()
	}()

	e.started.Store(true)

	// Read the items
	ch := watcher.ResultChan()
	for {
		// Prioritize checking for finished watcher
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		default:
		}

		// Wait for the results
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		case event, ok := <-ch:
			// Handle closed watcher
			if !ok {
				return e.ctx.Err()
			}

			// Load the current Kubernetes object
			object, ok := event.Object.(*batchv1.Job)
			if !ok || object == nil {
				continue
			}

			// Save the latest resource version to recover
			e.mu.Lock()
			e.opts.ResourceVersion = object.ResourceVersion
			e.mu.Unlock()

			// Continue watching if that's just a bookmark
			if event.Type == watch.Bookmark {
				continue
			}

			// Send the event back
			e.ch <- object
			e.setLastJob(object)

			// Handle the deletion
			e.mu.Lock()
			if e.finalize(object) {
				e.mu.Unlock()
				return ErrDone
			}
			e.mu.Unlock()
		}
	}
}

func (e *jobWatcher) cycle() {
	// Close the channel when the watcher is stopped
	go func() {
		<-e.ctx.Done()
		close(e.ch)

		e.peekMu.Lock()
		defer e.peekMu.Unlock()
		peekCh := e.peekCh
		e.peekCh = nil
		if peekCh != nil {
			close(e.peekCh)
		}
	}()

	// Read the initial data
	_, err := e.read(0)
	if err != nil {
		e.setError(err)
		return
	}

	// Watch for the data updates,
	// and restart the watcher as long as there are no errors
	for err == nil {
		err = e.watch()
	}
	e.setError(err)
	e.cancel()
}

func (e *jobWatcher) Peek(ctx context.Context) <-chan *batchv1.Job {
	ch := make(chan *batchv1.Job)

	go func() {
		select {
		case <-e.peekCh:
		case <-ctx.Done():
			close(ch)
			return
		}
		e.peekMu.Lock()
		job := e.peek
		e.peekMu.Unlock()
		if job != nil {
			ch <- job
		}
		close(ch)
	}()

	return ch
}

func (e *jobWatcher) Err() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.err != nil {
		return e.err
	}
	return e.ctx.Err()
}

func (e *jobWatcher) Done() <-chan struct{} {
	return e.ctx.Done()
}

// Channel returns the channel for reading the job.
func (e *jobWatcher) Channel() <-chan *batchv1.Job {
	return e.ch
}

// Stop cancels all the on-going communication
func (e *jobWatcher) Stop() {
	e.cancel()
}

// Update gets the latest list of the job, to ensure that nothing is missed at that point.
// It returns number of items that have been appended.
func (e *jobWatcher) Update(t time.Duration) (int, error) {
	count, err := e.read(t)
	if errors.Is(err, ErrDone) {
		err = nil
	}
	return count, err
}
