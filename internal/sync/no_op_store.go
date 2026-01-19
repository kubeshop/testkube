package sync

import (
	"context"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

// NoOpStore provides a fallback store implementation for when a concrete store
// cannot be used, for example when an appropriate gRPC server cannot be contacted.
// It performs no operations and returns no errors from all of its functions.
type NoOpStore struct{}

func (s NoOpStore) UpdateOrCreateTestTrigger(_ context.Context, _ testtriggersv1.TestTrigger) error {
	return nil
}

func (s NoOpStore) DeleteTestTrigger(_ context.Context, _ string) error {
	return nil
}

func (s NoOpStore) UpdateOrCreateTestWorkflow(_ context.Context, _ testworkflowsv1.TestWorkflow) error {
	return nil
}

func (s NoOpStore) DeleteTestWorkflow(_ context.Context, _ string) error {
	return nil
}

func (s NoOpStore) UpdateOrCreateTestWorkflowTemplate(_ context.Context, _ testworkflowsv1.TestWorkflowTemplate) error {
	return nil
}

func (s NoOpStore) DeleteTestWorkflowTemplate(_ context.Context, _ string) error {
	return nil
}

func (s NoOpStore) UpdateOrCreateWebhook(_ context.Context, _ executorv1.Webhook) error {
	return nil
}

func (s NoOpStore) DeleteWebhook(_ context.Context, _ string) error {
	return nil
}

func (s NoOpStore) UpdateOrCreateWebhookTemplate(_ context.Context, _ executorv1.WebhookTemplate) error {
	return nil
}

func (s NoOpStore) DeleteWebhookTemplate(_ context.Context, _ string) error {
	return nil
}
