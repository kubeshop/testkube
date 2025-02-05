package webhook

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

type CloudRepository struct {
	executor executor.Executor
}

//go:generate mockgen -destination=./mock_webhook.go -package=webhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook" WebhookRepository
type WebhookRepository interface {
	CollectExecutionTelemetry(ctx context.Context, event testkube.Event, webhookName, errorMessage string, statusCode int) error
}

func NewCloudRepository(cloudClient cloud.TestKubeCloudAPIClient, apiKey string) *CloudRepository {
	return &CloudRepository{executor: executor.NewCloudGRPCExecutor(cloudClient, apiKey)}
}

func (c *CloudRepository) CollectExecutionTelemetry(ctx context.Context, event testkube.Event, webhookName, errorMessage string, statusCode int) error {
	var executionID, workflowName string
	if event.TestWorkflowExecution != nil {
		executionID = event.TestWorkflowExecution.Id
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}
	}

	var eventType testkube.EventType
	if event.Type_ != nil {
		eventType = *event.Type_
	}

	req := WebhookExecutionCollectResultRequest{
		ExecutionID:  executionID,
		WorkflowName: workflowName,
		WebhookName:  webhookName,
		EventType:    eventType,
		ErrorMessage: errorMessage,
		StatusCode:   statusCode,
	}

	if _, err := c.executor.Execute(ctx, CmdWebhookExecutionCollectResult, req); err != nil {
		return err
	}

	return nil
}
