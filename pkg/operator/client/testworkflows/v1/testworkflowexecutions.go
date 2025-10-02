package v1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

//go:generate mockgen -destination=./mock_testworkflowexecutions.go -package=v1 "github.com/kubeshop/testkube/pkg/operator/client/testworkflows/v1" TestWorkflowExecutionsInterface
type TestWorkflowExecutionsInterface interface {
	Get(name string) (*testworkflowsv1.TestWorkflowExecution, error)
	Create(testWorkflowExecution *testworkflowsv1.TestWorkflowExecution) (*testworkflowsv1.TestWorkflowExecution, error)
	Update(testWorkflowExecution *testworkflowsv1.TestWorkflowExecution) (*testworkflowsv1.TestWorkflowExecution, error)
	Delete(name string) error
	UpdateStatus(testWorkflowЕxecution *testworkflowsv1.TestWorkflowExecution) error
}

// NewTestWorkflowExecutionsClient returns new client instance, needs kubernetes client to be passed as dependecy
func NewTestWorkflowExecutionsClient(client client.Client, namespace string) *TestWorkflowExecutionsClient {
	return &TestWorkflowExecutionsClient{
		k8sClient: client,
		namespace: namespace,
	}
}

// TestWorkflowExecutionsClient client for getting test workflow executions CRs
type TestWorkflowExecutionsClient struct {
	k8sClient client.Client
	namespace string
}

// Get gets test workflow execution by name in given namespace
func (s TestWorkflowExecutionsClient) Get(name string) (*testworkflowsv1.TestWorkflowExecution, error) {
	testWorkflowExecution := &testworkflowsv1.TestWorkflowExecution{}
	if err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: name}, testWorkflowExecution); err != nil {
		return nil, err
	}

	return testWorkflowExecution, nil
}

// Create creates new test workflow execution CRD
func (s TestWorkflowExecutionsClient) Create(testWorkflowExecution *testworkflowsv1.TestWorkflowExecution) (*testworkflowsv1.TestWorkflowExecution, error) {
	if err := s.k8sClient.Create(context.Background(), testWorkflowExecution); err != nil {
		return nil, err
	}

	return testWorkflowExecution, nil
}

// Update updates test workflow execution
func (s TestWorkflowExecutionsClient) Update(testWorkflowExecution *testworkflowsv1.TestWorkflowExecution) (*testworkflowsv1.TestWorkflowExecution, error) {
	if err := s.k8sClient.Update(context.Background(), testWorkflowExecution); err != nil {
		return nil, err
	}

	return testWorkflowExecution, nil
}

// Delete deletes test workflow execution by name
func (s TestWorkflowExecutionsClient) Delete(name string) error {
	testWorkflowExecution := &testworkflowsv1.TestWorkflowExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}

	err := s.k8sClient.Delete(context.Background(), testWorkflowExecution)
	return err
}

// UpdateStatus updates existing test workflow execution status
func (s TestWorkflowExecutionsClient) UpdateStatus(testWorkflowExecution *testworkflowsv1.TestWorkflowExecution) error {
	return s.k8sClient.Status().Update(context.Background(), testWorkflowExecution)
}
