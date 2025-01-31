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

type ListTestWorkflowOptions struct {
	Labels     map[string]string
	TextSearch string
	Offset     uint32
	Limit      uint32
}

type TestWorkflowUpdate struct {
	Type      cloud.UpdateType
	Timestamp time.Time
	Resource  *testkube.TestWorkflow
}

type TestWorkflowsReader channels.Watcher[*testkube.TestWorkflow]
type TestWorkflowWatcher channels.Watcher[*TestWorkflowUpdate]

type TestWorkflowsClient interface {
	GetTestWorkflow(ctx context.Context, environmentId, name string) (*testkube.TestWorkflow, error)
	ListTestWorkflows(ctx context.Context, environmentId string, options ListTestWorkflowOptions) TestWorkflowsReader
	ListTestWorkflowLabels(ctx context.Context, environmentId string) (map[string][]string, error)
	UpdateTestWorkflow(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error
	CreateTestWorkflow(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error
	DeleteTestWorkflow(ctx context.Context, environmentId, name string) error
	DeleteTestWorkflowsByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error)
	WatchTestWorkflowUpdates(ctx context.Context, environmentId string, includeInitialData bool) TestWorkflowWatcher
}

func (c *client) GetTestWorkflow(ctx context.Context, environmentId, name string) (*testkube.TestWorkflow, error) {
	req := &cloud.GetTestWorkflowRequest{Name: name}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetTestWorkflow, req)
	if err != nil {
		return nil, err
	}
	var workflow testkube.TestWorkflow
	if err = json.Unmarshal(res.Workflow, &workflow); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (c *client) ListTestWorkflows(ctx context.Context, environmentId string, options ListTestWorkflowOptions) TestWorkflowsReader {
	req := &cloud.ListTestWorkflowsRequest{
		Offset:     options.Offset,
		Limit:      100,
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListTestWorkflows, req)
	if err != nil {
		return channels.NewError[*testkube.TestWorkflow](err)
	}
	result := channels.NewWatcher[*testkube.TestWorkflow]()
	go func() {
		var item *cloud.TestWorkflowListItem
		for err == nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			var workflow testkube.TestWorkflow
			err = json.Unmarshal(item.Workflow, &workflow)
			result.Send(&workflow)
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		result.Close(err)
	}()
	return result
}

func (c *client) ListTestWorkflowLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	req := &cloud.ListTestWorkflowLabelsRequest{}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListTestWorkflowLabels, req)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(res.Labels))
	for _, label := range res.Labels {
		result[label.Name] = label.Value
	}
	return result, nil
}

func (c *client) UpdateTestWorkflow(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	req := &cloud.UpdateTestWorkflowRequest{Workflow: workflowBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.UpdateTestWorkflow, req)
	return err
}

func (c *client) CreateTestWorkflow(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	req := &cloud.CreateTestWorkflowRequest{Workflow: workflowBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.CreateTestWorkflow, req)
	return err
}

func (c *client) DeleteTestWorkflow(ctx context.Context, environmentId, name string) error {
	req := &cloud.DeleteTestWorkflowRequest{Name: name}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteTestWorkflow, req)
	return err
}

func (c *client) DeleteTestWorkflowsByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	req := &cloud.DeleteTestWorkflowsByLabelsRequest{Labels: labels}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteTestWorkflowsByLabels, req)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}

func (c *client) WatchTestWorkflowUpdates(ctx context.Context, environmentId string, includeInitialData bool) TestWorkflowWatcher {
	req := &cloud.WatchTestWorkflowUpdatesRequest{IncludeInitialData: includeInitialData}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.WatchTestWorkflowUpdates, req)
	if err != nil {
		return channels.NewError[*TestWorkflowUpdate](err)
	}
	watcher := channels.NewWatcher[*TestWorkflowUpdate]()
	go func() {
		var item *cloud.TestWorkflowUpdate
		for err == nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			if item.Ping {
				continue
			}
			var resource testkube.TestWorkflow
			err = json.Unmarshal(item.Resource, &resource)
			watcher.Send(&TestWorkflowUpdate{
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
