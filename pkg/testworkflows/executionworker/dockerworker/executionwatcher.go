package dockerworker

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"golang.org/x/sync/singleflight"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/log"
	store2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/store"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/ui"
)

type executionWatcher struct {
	id        string
	signature []stage.Signature

	client    *dockerclient.Client
	ctx       context.Context
	ctxCancel context.CancelFunc

	initialCommitCh          chan struct{}
	initialCommitInitialized atomic.Bool

	state       *executionState
	uncommitted *executionState
	update      store2.Update
	mu          sync.RWMutex
}

func NewExecutionWatcher(ctx context.Context, client *dockerclient.Client, id string, signature []stage.Signature, scheduledAt time.Time) watchers.ExecutionWatcher {
	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(ctx)

	// Build initial data
	opts := ExecutionStateOptions{
		ResourceId:  id,
		Signature:   signature,
		ScheduledAt: scheduledAt,
	}

	watcher := &executionWatcher{
		id:        id,
		signature: signature,

		client:          client,
		ctx:             ctx,
		ctxCancel:       ctxCancel,
		initialCommitCh: make(chan struct{}),

		state:       NewExecutionState(nil, &opts),
		uncommitted: NewExecutionState(nil, &opts),
		update:      store2.NewUpdate(),
	}

	go func() {
		if ctx.Err() != nil {
			return
		}
		list, err := client.ContainerList(ctx, container.ListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("%s=%s", constants.ResourceIdLabelName, id)}),
		})
		if ctx.Err() != nil {
			return
		}

		actual := make([]types.ContainerJSON, len(list))
		actualMu := sync.Mutex{}
		var wg sync.WaitGroup
		wg.Add(len(list))
		for i := range list {
			go func() {
				defer wg.Done()
				v, e := client.ContainerInspect(ctx, list[i].ID)
				if e != nil {
					log.DefaultLogger.Warnw("failed to inspect container", "error", e)
				} else {
					actual[i] = v
				}
			}()
		}
		wg.Wait()

		watcher.uncommitted = NewExecutionState(actual, &opts)
		watcher.Commit()

		messages, errs := client.Events(ctx, events.ListOptions{
			Filters: filters.NewArgs(
				filters.KeyValuePair{Key: "type", Value: "container"},
				filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("%s=%s", constants.ResourceIdLabelName, id)},
			),
		})

		var sf singleflight.Group

	loop:
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-errs:
				if !ok {
					break loop
				}
				fmt.Println(ui.Red(fmt.Sprintf("ERROR: %s:", err.Error())))
				// FIXME: fail or retry?
			case event, ok := <-messages:
				if !ok {
					break loop
				}
				go func() {
					sf.Do(id, func() (interface{}, error) {
						v, e := client.ContainerInspect(ctx, event.Actor.ID)
						if e != nil {
							log.DefaultLogger.Warnw("failed to inspect container", "error", e)
							return nil, e
						}
						actualMu.Lock()
						defer actualMu.Unlock()
						index := slices.IndexFunc(actual, func(json types.ContainerJSON) bool {
							return json.ID == v.ID
						})
						if index == -1 {
							actual = append(actual, v)
						} else {
							actual[index] = v
						}
						watcher.uncommitted = NewExecutionState(actual, &opts)
						watcher.Commit()
						return nil, nil
					})
				}()
			}
		}

		if err != nil {
			log.DefaultLogger.Warnw("failed to list containers", "error", err)
		}
	}()

	return watcher
}

func (e *executionWatcher) State() watchers.ExecutionState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

func (e *executionWatcher) Commit() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state = common.Ptr(*e.uncommitted)
	if e.initialCommitInitialized.CompareAndSwap(false, true) {
		close(e.initialCommitCh)
	}
	e.update.Emit()
}

func (e *executionWatcher) Refresh(ctx context.Context) {

}

func (e *executionWatcher) Started() <-chan struct{} {
	return e.initialCommitCh
}

func (e *executionWatcher) Updated(ctx context.Context) <-chan struct{} {
	return e.update.Channel(ctx)
}

func (e *executionWatcher) Next() <-chan struct{} {
	return e.update.Next()
}
