package crdstorage

import (
	"context"
	"fmt"
	"math"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
	"github.com/kubeshop/testkube/pkg/resourcepattern"
)

type testWorkflowsStorage struct {
	client        testworkflowclient.TestWorkflowClient
	environmentId string
	pattern       resourcepattern.Pattern
	metadata      map[string]string
}

func NewTestWorkflowsStorage(
	client testworkflowclient.TestWorkflowClient,
	environmentId string,
	pattern string,
	metadata map[string]string,
) (Storage[testkube.TestWorkflow], error) {
	patternUtil, err := resourcepattern.New(pattern)
	if err != nil {
		return nil, err
	}
	return &testWorkflowsStorage{
		client:        client,
		environmentId: environmentId,
		pattern:       patternUtil,
		metadata:      metadata,
	}, err
}

func (s *testWorkflowsStorage) Process(ctx context.Context, event Event[testkube.TestWorkflow]) error {
	// Convert the resource from one source to another
	name, ok := s.pattern.Compile(&event.Metadata)
	if !ok {
		// Don't process if it's not covered
		return nil
	}
	event.Resource.Name = name
	event.Resource.Namespace = event.Metadata.Generic["namespace"]

	// Decide on the action
	switch event.Type {
	case EventTypeCreate:
		// Avoid processing when there is no difference, or we have newer result
		current, err := s.client.Get(ctx, s.environmentId, name)
		if err == nil && (current.Equals(&event.Resource) || (!event.Timestamp.IsZero() && !current.Updated.Before(event.Timestamp))) {
			return nil
		}
		if current != nil {
			return s.client.Update(ctx, s.environmentId, event.Resource)
		}
		return s.client.Create(ctx, s.environmentId, event.Resource)
	case EventTypeUpdate:
		// Avoid processing when there is no difference, or we have newer result
		current, err := s.client.Get(ctx, s.environmentId, name)
		if err == nil && (current.Equals(&event.Resource) || (!event.Timestamp.IsZero() && !current.Updated.Before(event.Timestamp))) {
			return nil
		}
		if err != nil {
			return s.client.Create(ctx, s.environmentId, event.Resource)
		}
		return s.client.Update(ctx, s.environmentId, event.Resource)
	case EventTypeDelete:
		// Avoid processing when there is no difference, or we have newer result
		current, err := s.client.Get(ctx, s.environmentId, name)
		if err == nil && (!event.Timestamp.IsZero() && !current.Updated.Before(event.Timestamp)) {
			return nil
		}
		return s.client.Delete(ctx, s.environmentId, name)
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
}

func (s *testWorkflowsStorage) List(ctx context.Context) channels.Watcher[Resource[testkube.TestWorkflow]] {
	watcher := channels.NewWatcher[Resource[testkube.TestWorkflow]]()
	go func() {
		workflows, err := s.client.List(ctx, s.environmentId, testworkflowclient.ListOptions{
			Limit: math.MaxInt32,
		})
		if err != nil {
			watcher.Close(err)
			return
		}
		for _, workflow := range workflows {
			metadata, ok := s.pattern.Parse(workflow.Name, map[string]string{
				"namespace": workflow.Namespace,
			})
			if !ok {
				continue
			}
			watcher.Send(Resource[testkube.TestWorkflow]{Metadata: *metadata, Resource: workflow})
		}
		watcher.Close(nil)
	}()
	return watcher
}

func (s *testWorkflowsStorage) Watch(ctx context.Context) channels.Watcher[Event[testkube.TestWorkflow]] {
	return channels.Transform(s.client.WatchUpdates(ctx, s.environmentId, true), func(t testworkflowclient.Update) (Event[testkube.TestWorkflow], bool) {
		metadata, ok := s.pattern.Parse(t.Resource.Name, map[string]string{
			"namespace": t.Resource.Namespace,
		})
		if !ok {
			return Event[testkube.TestWorkflow]{}, false
		}
		ts := t.Timestamp
		if t.Resource != nil && t.Resource.Updated.After(ts) {
			ts = t.Resource.Updated
		}
		return Event[testkube.TestWorkflow]{
			Type:      EventType(t.Type),
			Timestamp: ts,
			Resource:  *t.Resource,
			Metadata:  *metadata,
		}, true
	})
}
