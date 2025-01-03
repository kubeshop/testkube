package testworkflowclient

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

var _ TestWorkflowClient = &cloudTestWorkflowClient{}

type cloudTestWorkflowClient struct {
	client cloud.TestKubeCloudAPIClient
	apiKey string
}

func NewCloudTestWorkflowClient(client cloud.TestKubeCloudAPIClient, apiKey string) TestWorkflowClient {
	return &cloudTestWorkflowClient{
		client: client,
		apiKey: apiKey,
	}
}

func (c *cloudTestWorkflowClient) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.GetTestWorkflow(ctx, &cloud.GetTestWorkflowRequest{
		EnvironmentId: environmentId,
		Name:          name,
	})
	if err != nil {
		return nil, err
	}

	var workflow testkube.TestWorkflow
	if err = json.Unmarshal(resp.Workflow, &workflow); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (c *cloudTestWorkflowClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflow, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.ListTestWorkflows(ctx, &cloud.ListTestWorkflowsRequest{
		EnvironmentId: environmentId,
		Offset:        options.Offset,
		Limit:         options.Limit,
		Labels:        options.Labels,
		TextSearch:    options.TextSearch,
	})
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflow, 0)
	var item *cloud.TestWorkflowListItem
	for {
		item, err = resp.Recv()
		if err != nil {
			break
		}
		var workflow testkube.TestWorkflow
		err = json.Unmarshal(item.Workflow, &workflow)
		if err != nil {
			return nil, err
		}
		result = append(result, workflow)
	}
	return result, err
}

func (c *cloudTestWorkflowClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.ListTestWorkflowLabels(ctx, &cloud.ListTestWorkflowLabelsRequest{
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

func (c *cloudTestWorkflowClient) Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	_, err = c.client.UpdateTestWorkflow(ctx, &cloud.UpdateTestWorkflowRequest{
		EnvironmentId: environmentId,
		Workflow:      workflowBytes,
	})
	return err
}

func (c *cloudTestWorkflowClient) Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	_, err = c.client.CreateTestWorkflow(ctx, &cloud.CreateTestWorkflowRequest{
		EnvironmentId: environmentId,
		Workflow:      workflowBytes,
	})
	return err
}

func (c *cloudTestWorkflowClient) Delete(ctx context.Context, environmentId string, name string) error {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	_, err := c.client.DeleteTestWorkflow(ctx, &cloud.DeleteTestWorkflowRequest{
		EnvironmentId: environmentId,
		Name:          name,
	})
	return err
}

func (c *cloudTestWorkflowClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey))

	resp, err := c.client.DeleteTestWorkflowsByLabels(ctx, &cloud.DeleteTestWorkflowsByLabelsRequest{
		EnvironmentId: environmentId,
		Labels:        labels,
	})
	if err != nil {
		return 0, err
	}
	return resp.Count, nil
}
