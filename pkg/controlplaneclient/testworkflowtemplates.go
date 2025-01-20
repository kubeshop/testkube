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

type ListTestWorkflowTemplateOptions struct {
	Labels     map[string]string
	TextSearch string
	Offset     uint32
	Limit      uint32
}

type TestWorkflowTemplatesReader channels.Watcher[*testkube.TestWorkflowTemplate]

type TestWorkflowTemplatesClient interface {
	GetTestWorkflowTemplate(ctx context.Context, environmentId, name string) (*testkube.TestWorkflowTemplate, error)
	ListTestWorkflowTemplates(ctx context.Context, environmentId string, options ListTestWorkflowTemplateOptions) TestWorkflowTemplatesReader
	ListTestWorkflowTemplateLabels(ctx context.Context, environmentId string) (map[string][]string, error)
	UpdateTestWorkflowTemplate(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error
	CreateTestWorkflowTemplate(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error
	DeleteTestWorkflowTemplate(ctx context.Context, environmentId, name string) error
	DeleteTestWorkflowTemplatesByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error)
}

func (c *client) GetTestWorkflowTemplate(ctx context.Context, environmentId, name string) (*testkube.TestWorkflowTemplate, error) {
	req := &cloud.GetTestWorkflowTemplateRequest{Name: name}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetTestWorkflowTemplate, req)
	if err != nil {
		return nil, err
	}
	var workflow testkube.TestWorkflowTemplate
	if err = json.Unmarshal(res.Template, &workflow); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (c *client) ListTestWorkflowTemplates(ctx context.Context, environmentId string, options ListTestWorkflowTemplateOptions) TestWorkflowTemplatesReader {
	req := &cloud.ListTestWorkflowTemplatesRequest{
		Offset:     options.Offset,
		Limit:      options.Limit,
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListTestWorkflowTemplates, req)
	if err != nil {
		return channels.NewError[*testkube.TestWorkflowTemplate](err)
	}

	result := channels.NewWatcher[*testkube.TestWorkflowTemplate]()
	go func() {
		var item *cloud.TestWorkflowTemplateListItem
		for err != nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			var workflow testkube.TestWorkflowTemplate
			err = json.Unmarshal(item.Template, &workflow)
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		result.Close(err)
	}()
	return result
}

func (c *client) ListTestWorkflowTemplateLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	req := &cloud.ListTestWorkflowTemplateLabelsRequest{}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListTestWorkflowTemplateLabels, req)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(res.Labels))
	for _, label := range res.Labels {
		result[label.Name] = label.Value
	}
	return result, nil
}

func (c *client) UpdateTestWorkflowTemplate(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error {
	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	req := &cloud.UpdateTestWorkflowTemplateRequest{Template: workflowBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.UpdateTestWorkflowTemplate, req)
	return err
}

func (c *client) CreateTestWorkflowTemplate(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error {
	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	req := &cloud.CreateTestWorkflowTemplateRequest{Template: workflowBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.CreateTestWorkflowTemplate, req)
	return err
}

func (c *client) DeleteTestWorkflowTemplate(ctx context.Context, environmentId, name string) error {
	req := &cloud.DeleteTestWorkflowTemplateRequest{Name: name}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteTestWorkflowTemplate, req)
	return err
}

func (c *client) DeleteTestWorkflowTemplatesByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	req := &cloud.DeleteTestWorkflowTemplatesByLabelsRequest{Labels: labels}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteTestWorkflowTemplatesByLabels, req)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}
