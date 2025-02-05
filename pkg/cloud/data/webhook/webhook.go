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
	CollectExecutionResult(ctx context.Context, event testkube.Event, webhookName, errorMessage string, statusCode int) error
}

func NewCloudRepository(cloudClient cloud.TestKubeCloudAPIClient, apiKey string) *CloudRepository {
	return &CloudRepository{executor: executor.NewCloudGRPCExecutor(cloudClient, apiKey)}
}

func (c *CloudRepository) CollectExecutionResult(ctx context.Context, event testkube.Event, webhookName, errorMessage string, statusCode int) error {
	var executionID, name string
	if event.TestExecution != nil {
		executionID = event.TestExecution.Id
		name = event.TestExecution.TestName
	}

	if event.TestSuiteExecution != nil {
		executionID = event.TestSuiteExecution.Id
		if event.TestSuiteExecution.TestSuite != nil {
			name = event.TestSuiteExecution.TestSuite.Name
		}
	}

	if event.TestWorkflowExecution != nil {
		executionID = event.TestWorkflowExecution.Id
		if event.TestWorkflowExecution.Workflow != nil {
			name = event.TestWorkflowExecution.Workflow.Name
		}
	}

	var eventType testkube.EventType
	if event.Type_ != nil {
		eventType = *event.Type_
	}

	req := WebhookExecutionCollectResultRequest{
		ExecutionID:  executionID,
		Name:         name,
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
