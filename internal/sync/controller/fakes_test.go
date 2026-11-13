package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

var fakeNotFoundErr = errors.NewNotFound(schema.GroupResource{}, "test-error")

// testEnvironments maps the namespaces of the tests to their Environments; any other namespace
// syncs into the default Environment.
var testEnvironments = map[string]string{"team-b": "env-b", "team-c": "env-b"}

// testTarget is the TargetFunc of the tests.
func testTarget(namespace string) syncagent.Target {
	return syncagent.Target{Namespace: namespace, EnvironmentID: testEnvironments[namespace]}
}

type fakeKubernetesClient struct {
	Err                  error
	TestTrigger          testtriggersv1.TestTrigger
	WorkflowTrigger      workflowtriggersv1.WorkflowTrigger
	TestWorkflow         testworkflowsv1.TestWorkflow
	TestWorkflowTemplate testworkflowsv1.TestWorkflowTemplate
	Webhook              executorv1.Webhook
	WebhookTemplate      executorv1.WebhookTemplate

	// ListErr is returned by List, and the slices below are what List returns for each kind.
	ListErr               error
	TestTriggers          []testtriggersv1.TestTrigger
	WorkflowTriggers      []workflowtriggersv1.WorkflowTrigger
	TestWorkflows         []testworkflowsv1.TestWorkflow
	TestWorkflowTemplates []testworkflowsv1.TestWorkflowTemplate
	Webhooks              []executorv1.Webhook
	WebhookTemplates      []executorv1.WebhookTemplate
}

func (t fakeKubernetesClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch v := obj.(type) {
	case *testtriggersv1.TestTrigger:
		t.TestTrigger.DeepCopyInto(v)
	case *workflowtriggersv1.WorkflowTrigger:
		t.WorkflowTrigger.DeepCopyInto(v)
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

func (t fakeKubernetesClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	switch v := list.(type) {
	case *testtriggersv1.TestTriggerList:
		v.Items = append(v.Items[:0], t.TestTriggers...)
	case *workflowtriggersv1.WorkflowTriggerList:
		v.Items = append(v.Items[:0], t.WorkflowTriggers...)
	case *testworkflowsv1.TestWorkflowList:
		v.Items = append(v.Items[:0], t.TestWorkflows...)
	case *testworkflowsv1.TestWorkflowTemplateList:
		v.Items = append(v.Items[:0], t.TestWorkflowTemplates...)
	case *executorv1.WebhookList:
		v.Items = append(v.Items[:0], t.Webhooks...)
	case *executorv1.WebhookTemplateList:
		v.Items = append(v.Items[:0], t.WebhookTemplates...)
	}
	return t.ListErr
}

type fakeStore struct {
	TestTrigger          testtriggersv1.TestTrigger
	WorkflowTrigger      workflowtriggersv1.WorkflowTrigger
	TestWorkflow         testworkflowsv1.TestWorkflow
	TestWorkflowTemplate testworkflowsv1.TestWorkflowTemplate
	Webhook              executorv1.Webhook
	WebhookTemplate      executorv1.WebhookTemplate
	Deleted              string
	// Target is the target of the last call.
	Target      syncagent.Target
	UpdateCalls int
	// Err is returned by every store method, to exercise the reconcilers' error paths.
	Err error
}

func (t *fakeStore) UpdateOrCreateTestTrigger(_ context.Context, target syncagent.Target, trigger testtriggersv1.TestTrigger) error {
	t.Target = target
	t.UpdateCalls++
	trigger.DeepCopyInto(&t.TestTrigger)
	return t.Err
}

func (t *fakeStore) DeleteTestTrigger(_ context.Context, target syncagent.Target, s string) error {
	t.Target = target
	t.Deleted = s
	return t.Err
}

func (t *fakeStore) UpdateOrCreateWorkflowTrigger(_ context.Context, target syncagent.Target, trigger workflowtriggersv1.WorkflowTrigger) error {
	t.Target = target
	t.UpdateCalls++
	trigger.DeepCopyInto(&t.WorkflowTrigger)
	return t.Err
}

func (t *fakeStore) DeleteWorkflowTrigger(_ context.Context, target syncagent.Target, s string) error {
	t.Target = target
	t.Deleted = s
	return t.Err
}

func (t *fakeStore) UpdateOrCreateTestWorkflow(_ context.Context, target syncagent.Target, workflow testworkflowsv1.TestWorkflow) error {
	t.Target = target
	t.UpdateCalls++
	workflow.DeepCopyInto(&t.TestWorkflow)
	return t.Err
}

func (t *fakeStore) DeleteTestWorkflow(_ context.Context, target syncagent.Target, s string) error {
	t.Target = target
	t.Deleted = s
	return t.Err
}

func (t *fakeStore) UpdateOrCreateTestWorkflowTemplate(_ context.Context, target syncagent.Target, template testworkflowsv1.TestWorkflowTemplate) error {
	t.Target = target
	t.UpdateCalls++
	template.DeepCopyInto(&t.TestWorkflowTemplate)
	return t.Err
}

func (t *fakeStore) DeleteTestWorkflowTemplate(_ context.Context, target syncagent.Target, s string) error {
	t.Target = target
	t.Deleted = s
	return t.Err
}

func (t *fakeStore) UpdateOrCreateWebhook(_ context.Context, target syncagent.Target, webhook executorv1.Webhook) error {
	t.Target = target
	t.UpdateCalls++
	webhook.DeepCopyInto(&t.Webhook)
	return t.Err
}

func (t *fakeStore) DeleteWebhook(_ context.Context, target syncagent.Target, s string) error {
	t.Target = target
	t.Deleted = s
	return t.Err
}

func (t *fakeStore) UpdateOrCreateWebhookTemplate(_ context.Context, target syncagent.Target, template executorv1.WebhookTemplate) error {
	t.Target = target
	t.UpdateCalls++
	template.DeepCopyInto(&t.WebhookTemplate)
	return t.Err
}

func (t *fakeStore) DeleteWebhookTemplate(_ context.Context, target syncagent.Target, s string) error {
	t.Target = target
	t.Deleted = s
	return t.Err
}
