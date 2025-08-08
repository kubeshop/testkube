package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

type ListTestTriggerOptions struct {
	Labels     map[string]string
	TextSearch string
	Selector   string
	Offset     uint32
	Limit      uint32
}

type TestTriggerUpdate struct {
	Type      cloud.UpdateType
	Timestamp time.Time
	Resource  *testkube.TestTrigger
}

type TestTriggersReader channels.Watcher[*testkube.TestTrigger]
type TestTriggerWatcher channels.Watcher[*TestTriggerUpdate]

type TestTriggersClient interface {
	GetTestTrigger(ctx context.Context, environmentId, name, namespace string) (*testkube.TestTrigger, error)
	ListTestTriggers(ctx context.Context, environmentId string, options ListTestTriggerOptions, namespace string) TestTriggersReader
	ListTestTriggerLabels(ctx context.Context, environmentId, namespace string) (map[string][]string, error)
	UpdateTestTrigger(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error
	CreateTestTrigger(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error
	DeleteTestTrigger(ctx context.Context, environmentId, name, namespace string) error
	DeleteAllTestTriggers(ctx context.Context, environmentId, namespace string) (uint32, error)
	DeleteTestTriggersByLabels(ctx context.Context, environmentId, selector, namespace string) (uint32, error)
	WatchTestTriggerUpdates(ctx context.Context, environmentId, namespace string, includeInitialData bool) TestTriggerWatcher
}

func (c *client) GetTestTrigger(ctx context.Context, environmentId, name, namespace string) (*testkube.TestTrigger, error) {
	req := &cloud.GetTestTriggerRequest{Name: name, Namespace: namespace}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetTestTrigger, req)
	if err != nil {
		return nil, err
	}
	var trigger testkube.TestTrigger
	if err = json.Unmarshal(res.Trigger, &trigger); err != nil {
		return nil, err
	}
	return &trigger, nil
}

func (c *client) ListTestTriggers(ctx context.Context, environmentId string, options ListTestTriggerOptions, namespace string) TestTriggersReader {
	req := &cloud.ListTestTriggersRequest{
		Offset:     options.Offset,
		Limit:      options.Limit,
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Selector:   options.Selector,
		Namespace:  namespace,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListTestTriggers, req)
	if err != nil {
		return channels.NewError[*testkube.TestTrigger](err)
	}
	result := channels.NewWatcher[*testkube.TestTrigger]()
	go func() {
		var item *cloud.TestTriggerListItem
		for err == nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			var trigger testkube.TestTrigger
			err = json.Unmarshal(item.Trigger, &trigger)
			result.Send(&trigger)
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		result.Close(err)
	}()
	return result
}

func (c *client) ListTestTriggerLabels(ctx context.Context, environmentId, namespace string) (map[string][]string, error) {
	req := &cloud.ListTestTriggerLabelsRequest{Namespace: namespace}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListTestTriggerLabels, req)
	if err != nil {
		return nil, err
	}

	labels := make(map[string][]string)
	for _, label := range res.Labels {
		labels[label.Name] = label.Value
	}
	return labels, nil
}

func (c *client) CreateTestTrigger(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error {
	triggerBytes, err := json.Marshal(trigger)
	if err != nil {
		return err
	}
	req := &cloud.CreateTestTriggerRequest{Trigger: triggerBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.CreateTestTrigger, req)
	return err
}

func (c *client) UpdateTestTrigger(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error {
	triggerBytes, err := json.Marshal(trigger)
	if err != nil {
		return err
	}
	req := &cloud.UpdateTestTriggerRequest{Trigger: triggerBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.UpdateTestTrigger, req)
	return err
}

func (c *client) DeleteTestTrigger(ctx context.Context, environmentId, name, namespace string) error {
	req := &cloud.DeleteTestTriggerRequest{Name: name, Namespace: namespace}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteTestTrigger, req)
	return err
}

func (c *client) DeleteAllTestTriggers(ctx context.Context, environmentId, namespace string) (uint32, error) {
	req := &cloud.DeleteAllTestTriggersRequest{Namespace: namespace}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteAllTestTriggers, req)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}

func (c *client) DeleteTestTriggersByLabels(ctx context.Context, environmentId, selector, namespace string) (uint32, error) {
	req := &cloud.DeleteTestTriggersByLabelsRequest{Selector: selector, Namespace: namespace}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteTestTriggersByLabels, req)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}

func (c *client) WatchTestTriggerUpdates(ctx context.Context, environmentId, namespace string, includeInitialData bool) TestTriggerWatcher {
	req := &cloud.WatchTestTriggerUpdatesRequest{
		IncludeInitialData: includeInitialData,
		Namespace:          namespace,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.WatchTestTriggerUpdates, req)
	if err != nil {
		return channels.NewError[*TestTriggerUpdate](err)
	}
	watcher := channels.NewWatcher[*TestTriggerUpdate]()
	go func() {
		var item *cloud.TestTriggerUpdate
		for err == nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			if item.Ping {
				continue
			}
			var resource testkube.TestTrigger
			err = json.Unmarshal(item.Resource, &resource)
			watcher.Send(&TestTriggerUpdate{
				Type:      item.Type,
				Timestamp: item.Timestamp.AsTime(),
				Resource:  &resource,
			})
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		watcher.Close(err)
	}()
	return watcher
}
