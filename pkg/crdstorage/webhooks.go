package crdstorage

import (
	"context"
	"fmt"
	"math"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/webhookclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
	"github.com/kubeshop/testkube/pkg/resourcepattern"
)

type webhooksStorage struct {
	client        webhookclient.WebhookClient
	environmentId string
	pattern       resourcepattern.Pattern
	metadata      map[string]string
}

func NewWebhooksStorage(
	client webhookclient.WebhookClient,
	environmentId string,
	pattern string,
	metadata map[string]string,
) (Storage[testkube.Webhook], error) {
	patternUtil, err := resourcepattern.New(pattern)
	if err != nil {
		return nil, err
	}
	return &webhooksStorage{
		client:        client,
		environmentId: environmentId,
		pattern:       patternUtil,
		metadata:      metadata,
	}, err
}

func (s *webhooksStorage) Process(ctx context.Context, event Event[testkube.Webhook]) error {
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

func (s *webhooksStorage) List(ctx context.Context) channels.Watcher[Resource[testkube.Webhook]] {
	watcher := channels.NewWatcher[Resource[testkube.Webhook]]()
	go func() {
		webhooks, err := s.client.List(ctx, s.environmentId, webhookclient.ListOptions{
			Limit: math.MaxInt32,
		})
		if err != nil {
			watcher.Close(err)
			return
		}
		for _, webhook := range webhooks {
			metadata, ok := s.pattern.Parse(webhook.Name, map[string]string{
				"namespace": webhook.Namespace,
			})
			if !ok {
				continue
			}
			watcher.Send(Resource[testkube.Webhook]{Metadata: *metadata, Resource: webhook})
		}
		watcher.Close(nil)
	}()
	return watcher
}

func (s *webhooksStorage) Watch(ctx context.Context) channels.Watcher[Event[testkube.Webhook]] {
	return channels.Transform(s.client.WatchUpdates(ctx, s.environmentId, true), func(t webhookclient.Update) (Event[testkube.Webhook], bool) {
		metadata, ok := s.pattern.Parse(t.Resource.Name, map[string]string{
			"namespace": t.Resource.Namespace,
		})
		if !ok {
			return Event[testkube.Webhook]{}, false
		}
		ts := t.Timestamp
		if t.Resource != nil && t.Resource.Updated.After(ts) {
			ts = t.Resource.Updated
		}
		return Event[testkube.Webhook]{
			Type:      EventType(t.Type),
			Timestamp: ts,
			Resource:  *t.Resource,
			Metadata:  *metadata,
		}, true
	})
}
