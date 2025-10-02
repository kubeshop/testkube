package v1

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

//go:generate mockgen -destination=./mock_testworkflows.go -package=v1 "github.com/kubeshop/testkube/pkg/operator/client/testworkflows/v1" Interface
type Interface interface {
	List(selector string) (*testworkflowsv1.TestWorkflowList, error)
	ListLabels() (map[string][]string, error)
	Get(name string) (*testworkflowsv1.TestWorkflow, error)
	Create(workflow *testworkflowsv1.TestWorkflow) (*testworkflowsv1.TestWorkflow, error)
	Update(workflow *testworkflowsv1.TestWorkflow) (*testworkflowsv1.TestWorkflow, error)
	Apply(workflow *testworkflowsv1.TestWorkflow) error
	Delete(name string) error
	DeleteAll() error
	DeleteByLabels(selector string) error
	UpdateStatus(workflow *testworkflowsv1.TestWorkflow) error
}

// NewClient creates new TestWorkflow client
func NewClient(client client.Client, namespace string) *TestWorkflowsClient {
	return &TestWorkflowsClient{
		Client:    client,
		Namespace: namespace,
	}
}

// TestWorkflowsClient implements methods to work with TestWorkflows
type TestWorkflowsClient struct {
	Client    client.Client
	Namespace string
}

// List lists TestWorkflows
func (s TestWorkflowsClient) List(selector string) (*testworkflowsv1.TestWorkflowList, error) {
	list := &testworkflowsv1.TestWorkflowList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		Namespace:     s.Namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}

	if err = s.Client.List(context.Background(), list, options); err != nil {
		return list, err
	}

	return list, nil
}

// ListLabels lists labels for TestWorkflows
func (s TestWorkflowsClient) ListLabels() (map[string][]string, error) {
	labels := map[string][]string{}
	list := &testworkflowsv1.TestWorkflowList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	if err != nil {
		return labels, err
	}

	for _, workflow := range list.Items {
		for key, value := range workflow.Labels {
			if values, ok := labels[key]; !ok {
				labels[key] = []string{value}
			} else {
				for _, v := range values {
					if v == value {
						continue
					}
				}
				labels[key] = append(labels[key], value)
			}
		}
	}

	return labels, nil
}

// Get returns TestWorkflow
func (s TestWorkflowsClient) Get(name string) (*testworkflowsv1.TestWorkflow, error) {
	workflow := &testworkflowsv1.TestWorkflow{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, workflow)
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

// Create creates new TestWorkflow
func (s TestWorkflowsClient) Create(workflow *testworkflowsv1.TestWorkflow) (*testworkflowsv1.TestWorkflow, error) {
	return workflow, s.Client.Create(context.Background(), workflow)
}

// Update updates existing TestWorkflow
func (s TestWorkflowsClient) Update(workflow *testworkflowsv1.TestWorkflow) (*testworkflowsv1.TestWorkflow, error) {
	return workflow, s.Client.Update(context.Background(), workflow)
}

// Apply applies changes to the existing TestWorkflow
func (s TestWorkflowsClient) Apply(workflow *testworkflowsv1.TestWorkflow) error {
	return s.Client.Patch(context.Background(), workflow, client.Apply, &client.PatchOptions{
		FieldManager: "application/apply-patch",
	})
}

// Delete deletes existing TestWorkflow
func (s TestWorkflowsClient) Delete(name string) error {
	workflow, err := s.Get(name)
	if err != nil {
		return err
	}
	return s.Client.Delete(context.Background(), workflow)
}

// DeleteAll delete all TestWorkflows
func (s TestWorkflowsClient) DeleteAll() error {
	u := &unstructured.Unstructured{}
	u.SetKind("TestWorkflow")
	u.SetAPIVersion(testworkflowsv1.GroupVersion.String())
	return s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace))
}

// UpdateStatus updates existing TestWorkflow status
func (s TestWorkflowsClient) UpdateStatus(workflow *testworkflowsv1.TestWorkflow) error {
	return s.Client.Status().Update(context.Background(), workflow)
}

// DeleteByLabels deletes TestWorkflows by labels
func (s TestWorkflowsClient) DeleteByLabels(selector string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("TestWorkflow")
	u.SetAPIVersion(testworkflowsv1.GroupVersion.String())
	err = s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}
