package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

// ListWorkflowTriggerOptions mirrors the cloud.ListWorkflowTriggersRequest fields
// that the agent passes through when polling. Kept separate from the proto type
// so callers don't have to import the generated package.
type ListWorkflowTriggerOptions struct {
	Labels     map[string]string
	TextSearch string
	Selector   string
	Offset     uint32
	Limit      uint32
}

type WorkflowTriggersReader channels.Watcher[*testkube.WorkflowTrigger]

type WorkflowTriggersClient interface {
	GetWorkflowTrigger(ctx context.Context, environmentId, name, namespace string) (*testkube.WorkflowTrigger, error)
	ListWorkflowTriggers(ctx context.Context, environmentId string, options ListWorkflowTriggerOptions, namespace string) WorkflowTriggersReader
	UpdateWorkflowTrigger(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error
	CreateWorkflowTrigger(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error
	DeleteWorkflowTrigger(ctx context.Context, environmentId, name, namespace string) error
	DeleteAllWorkflowTriggers(ctx context.Context, environmentId, namespace string) (uint32, error)
	DeleteWorkflowTriggersByLabels(ctx context.Context, environmentId, selector, namespace string) (uint32, error)
}

func (c *client) GetWorkflowTrigger(ctx context.Context, environmentId, name, namespace string) (*testkube.WorkflowTrigger, error) {
	req := &cloud.GetWorkflowTriggerRequest{Name: name, Namespace: namespace}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetWorkflowTrigger, req)
	if err != nil {
		return nil, err
	}
	var trigger testkube.WorkflowTrigger
	if err = json.Unmarshal(res.Trigger, &trigger); err != nil {
		return nil, err
	}
	return &trigger, nil
}

func (c *client) ListWorkflowTriggers(ctx context.Context, environmentId string, options ListWorkflowTriggerOptions, namespace string) WorkflowTriggersReader {
	req := &cloud.ListWorkflowTriggersRequest{
		Offset:     options.Offset,
		Limit:      options.Limit,
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Selector:   options.Selector,
		Namespace:  namespace,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListWorkflowTriggers, req)
	if err != nil {
		return channels.NewError[*testkube.WorkflowTrigger](err)
	}
	result := channels.NewWatcher[*testkube.WorkflowTrigger]()
	go func() {
		var item *cloud.WorkflowTriggerListItem
		for err == nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			var trigger testkube.WorkflowTrigger
			err = json.Unmarshal(item.Trigger, &trigger)
			if err != nil {
				break
			}
			result.Send(&trigger)
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		result.Close(err)
	}()
	return result
}

func (c *client) CreateWorkflowTrigger(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error {
	triggerBytes, err := json.Marshal(trigger)
	if err != nil {
		return err
	}
	req := &cloud.CreateWorkflowTriggerRequest{Trigger: triggerBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.CreateWorkflowTrigger, req)
	return err
}

func (c *client) UpdateWorkflowTrigger(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error {
	triggerBytes, err := json.Marshal(trigger)
	if err != nil {
		return err
	}
	req := &cloud.UpdateWorkflowTriggerRequest{Trigger: triggerBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.UpdateWorkflowTrigger, req)
	return err
}

func (c *client) DeleteWorkflowTrigger(ctx context.Context, environmentId, name, namespace string) error {
	req := &cloud.DeleteWorkflowTriggerRequest{Name: name, Namespace: namespace}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteWorkflowTrigger, req)
	return err
}

func (c *client) DeleteAllWorkflowTriggers(ctx context.Context, environmentId, namespace string) (uint32, error) {
	req := &cloud.DeleteAllWorkflowTriggersRequest{Namespace: namespace}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteAllWorkflowTriggers, req)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}

func (c *client) DeleteWorkflowTriggersByLabels(ctx context.Context, environmentId, selector, namespace string) (uint32, error) {
	req := &cloud.DeleteWorkflowTriggersByLabelsRequest{Selector: selector, Namespace: namespace}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteWorkflowTriggersByLabels, req)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}
