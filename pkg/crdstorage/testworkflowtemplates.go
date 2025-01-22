package crdstorage

import (
	"context"
	"fmt"
	"math"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
	"github.com/kubeshop/testkube/pkg/resourcepattern"
)

type testWorkflowTemplatesStorage struct {
	client        testworkflowtemplateclient.TestWorkflowTemplateClient
	environmentId string
	pattern       resourcepattern.Pattern
	metadata      map[string]string
}

func NewTestWorkflowTemplatesStorage(
	client testworkflowtemplateclient.TestWorkflowTemplateClient,
	environmentId string,
	pattern string,
	metadata map[string]string,
) (Storage[testkube.TestWorkflowTemplate], error) {
	patternUtil, err := resourcepattern.New(pattern)
	if err != nil {
		return nil, err
	}
	return &testWorkflowTemplatesStorage{
		client:        client,
		environmentId: environmentId,
		pattern:       patternUtil,
		metadata:      metadata,
	}, err
}

func (s *testWorkflowTemplatesStorage) Process(ctx context.Context, event Event[testkube.TestWorkflowTemplate]) error {
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
		// Avoid processing when there is no difference
		current, err := s.client.Get(ctx, s.environmentId, name)
		if err == nil && (current.Equals(&event.Resource) || (!event.Timestamp.IsZero() && !current.Updated.Before(event.Timestamp))) {
			return nil
		}
		if current != nil {
			return s.client.Update(ctx, s.environmentId, event.Resource)
		}
		return s.client.Create(ctx, s.environmentId, event.Resource)
	case EventTypeUpdate:
		// Avoid processing when there is no difference
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

func (s *testWorkflowTemplatesStorage) List(ctx context.Context) channels.Watcher[Resource[testkube.TestWorkflowTemplate]] {
	watcher := channels.NewWatcher[Resource[testkube.TestWorkflowTemplate]]()
	go func() {
		templates, err := s.client.List(ctx, s.environmentId, testworkflowtemplateclient.ListOptions{
			Limit: math.MaxInt32,
		})
		if err != nil {
			watcher.Close(err)
			return
		}
		for _, template := range templates {
			metadata, ok := s.pattern.Parse(template.Name, map[string]string{
				"namespace": template.Namespace,
			})
			if !ok {
				continue
			}
			watcher.Send(Resource[testkube.TestWorkflowTemplate]{Metadata: *metadata, Resource: template})
		}
		watcher.Close(nil)
	}()
	return watcher
}

func (s *testWorkflowTemplatesStorage) Watch(ctx context.Context) channels.Watcher[Event[testkube.TestWorkflowTemplate]] {
	return channels.Transform(s.client.WatchUpdates(ctx, s.environmentId, true), func(t testworkflowtemplateclient.Update) (Event[testkube.TestWorkflowTemplate], bool) {
		metadata, ok := s.pattern.Parse(t.Resource.Name, map[string]string{
			"namespace": t.Resource.Namespace,
		})
		if !ok {
			return Event[testkube.TestWorkflowTemplate]{}, false
		}
		return Event[testkube.TestWorkflowTemplate]{
			Type:      EventType(t.Type),
			Timestamp: t.Timestamp,
			Resource:  *t.Resource,
			Metadata:  *metadata,
		}, true
	})
}
