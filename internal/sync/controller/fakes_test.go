package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

var fakeNotFoundErr = errors.NewNotFound(schema.GroupResource{}, "test-error")

type fakeKubernetesClient struct {
	Err                  error
	TestTrigger          testtriggersv1.TestTrigger
	TestWorkflow         testworkflowsv1.TestWorkflow
	TestWorkflowTemplate testworkflowsv1.TestWorkflowTemplate
	Webhook              executorv1.Webhook
	WebhookTemplate      executorv1.WebhookTemplate
}

func (t fakeKubernetesClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch v := obj.(type) {
	case *testtriggersv1.TestTrigger:
		t.TestTrigger.DeepCopyInto(v)
	case *testworkflowsv1.TestWorkflow:
		t.TestWorkflow.DeepCopyInto(v)
	case *testworkflowsv1.TestWorkflowTemplate:
		t.TestWorkflowTemplate.DeepCopyInto(v)
	case *executorv1.Webhook:
		t.Webhook.DeepCopyInto(v)
	case *executorv1.WebhookTemplate:
		t.WebhookTemplate.DeepCopyInto(v)
	}
	return t.Err
}

func (t fakeKubernetesClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}

type fakeStore struct {
	TestTrigger          testtriggersv1.TestTrigger
	TestWorkflow         testworkflowsv1.TestWorkflow
	TestWorkflowTemplate testworkflowsv1.TestWorkflowTemplate
	Webhook              executorv1.Webhook
	WebhookTemplate      executorv1.WebhookTemplate
	Deleted              string
	UpdateCalls          int
}

func (t *fakeStore) UpdateOrCreateTestTrigger(_ context.Context, trigger testtriggersv1.TestTrigger) error {
	t.UpdateCalls++
	trigger.DeepCopyInto(&t.TestTrigger)
	return nil
}

func (t *fakeStore) DeleteTestTrigger(_ context.Context, s string) error {
	t.Deleted = s
	return nil
}

func (t *fakeStore) UpdateOrCreateTestWorkflow(_ context.Context, workflow testworkflowsv1.TestWorkflow) error {
	t.UpdateCalls++
	workflow.DeepCopyInto(&t.TestWorkflow)
	return nil
}

func (t *fakeStore) DeleteTestWorkflow(_ context.Context, s string) error {
	t.Deleted = s
	return nil
}

func (t *fakeStore) UpdateOrCreateTestWorkflowTemplate(_ context.Context, template testworkflowsv1.TestWorkflowTemplate) error {
	t.UpdateCalls++
	template.DeepCopyInto(&t.TestWorkflowTemplate)
	return nil
}

func (t *fakeStore) DeleteTestWorkflowTemplate(_ context.Context, s string) error {
	t.Deleted = s
	return nil
}

func (t *fakeStore) UpdateOrCreateWebhook(_ context.Context, webhook executorv1.Webhook) error {
	t.UpdateCalls++
	webhook.DeepCopyInto(&t.Webhook)
	return nil
}

func (t *fakeStore) DeleteWebhook(_ context.Context, s string) error {
	t.Deleted = s
	return nil
}

func (t *fakeStore) UpdateOrCreateWebhookTemplate(_ context.Context, template executorv1.WebhookTemplate) error {
	t.UpdateCalls++
	template.DeepCopyInto(&t.WebhookTemplate)
	return nil
}

func (t *fakeStore) DeleteWebhookTemplate(_ context.Context, s string) error {
	t.Deleted = s
	return nil
}
