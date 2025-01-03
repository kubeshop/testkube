package testworkflowtemplateclient

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

var _ TestWorkflowTemplateClient = &cloudTestWorkflowTemplateClient{}

type cloudTestWorkflowTemplateClient struct {
	client cloud.TestKubeCloudAPIClient
	apiKey string
}

func NewCloudTestWorkflowTemplateClient(client cloud.TestKubeCloudAPIClient, apiKey string) TestWorkflowTemplateClient {
	return &cloudTestWorkflowTemplateClient{
		client: client,
		apiKey: apiKey,
	}
}

func (c *cloudTestWorkflowTemplateClient) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflowTemplate, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.GetTestWorkflowTemplate(ctx, &cloud.GetTestWorkflowTemplateRequest{
		EnvironmentId: environmentId,
		Name:          name,
	})
	if err != nil {
		return nil, err
	}

	var template testkube.TestWorkflowTemplate
	if err = json.Unmarshal(resp.Template, &template); err != nil {
		return nil, err
	}
	return &template, nil
}

func (c *cloudTestWorkflowTemplateClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflowTemplate, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.ListTestWorkflowTemplates(ctx, &cloud.ListTestWorkflowTemplatesRequest{
		EnvironmentId: environmentId,
		Offset:        options.Offset,
		Limit:         options.Limit,
		Labels:        options.Labels,
		TextSearch:    options.TextSearch,
	})
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowTemplate, 0)
	var item *cloud.TestWorkflowTemplateListItem
	for {
		item, err = resp.Recv()
		if err != nil {
			break
		}
		var template testkube.TestWorkflowTemplate
		err = json.Unmarshal(item.Template, &template)
		if err != nil {
			return nil, err
		}
		result = append(result, template)
	}
	return result, err
}

func (c *cloudTestWorkflowTemplateClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.ListTestWorkflowTemplateLabels(ctx, &cloud.ListTestWorkflowTemplateLabelsRequest{
		EnvironmentId: environmentId,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(resp.Labels))
	for _, label := range resp.Labels {
		result[label.Name] = label.Value
	}
	return result, nil
}

func (c *cloudTestWorkflowTemplateClient) Update(ctx context.Context, environmentId string, template testkube.TestWorkflowTemplate) error {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	templateBytes, err := json.Marshal(template)
	if err != nil {
		return err
	}
	_, err = c.client.UpdateTestWorkflowTemplate(ctx, &cloud.UpdateTestWorkflowTemplateRequest{
		EnvironmentId: environmentId,
		Template:      templateBytes,
	})
	return err
}

func (c *cloudTestWorkflowTemplateClient) Create(ctx context.Context, environmentId string, template testkube.TestWorkflowTemplate) error {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	templateBytes, err := json.Marshal(template)
	if err != nil {
		return err
	}
	_, err = c.client.CreateTestWorkflowTemplate(ctx, &cloud.CreateTestWorkflowTemplateRequest{
		EnvironmentId: environmentId,
		Template:      templateBytes,
	})
	return err
}

func (c *cloudTestWorkflowTemplateClient) Delete(ctx context.Context, environmentId string, name string) error {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	_, err := c.client.DeleteTestWorkflowTemplate(ctx, &cloud.DeleteTestWorkflowTemplateRequest{
		EnvironmentId: environmentId,
		Name:          name,
	})
	return err
}

func (c *cloudTestWorkflowTemplateClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.DeleteTestWorkflowTemplatesByLabels(ctx, &cloud.DeleteTestWorkflowTemplatesByLabelsRequest{
		EnvironmentId: environmentId,
		Labels:        labels,
	})
	if err != nil {
		return 0, err
	}
	return resp.Count, nil
}
