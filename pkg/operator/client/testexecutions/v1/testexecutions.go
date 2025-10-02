package testexecutions

import (
	"context"

	testexecutionv1 "github.com/kubeshop/testkube/api/testexecution/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -destination=./mock_testexecutions.go -package=testexecutions "github.com/kubeshop/testkube/pkg/operator/client/testexecutions/v1" Interface
type Interface interface {
	Get(name string) (*testexecutionv1.TestExecution, error)
	Create(testExecution *testexecutionv1.TestExecution) (*testexecutionv1.TestExecution, error)
	Update(testExecution *testexecutionv1.TestExecution) (*testexecutionv1.TestExecution, error)
	Delete(name string) error
	UpdateStatus(testЕxecution *testexecutionv1.TestExecution) error
}

// Option contain test execution options
type Option struct {
	Secrets map[string]string
}

// NewClient returns new client instance, needs kubernetes client to be passed as dependecy
func NewClient(client client.Client, namespace string) *TestExecutionsClient {
	return &TestExecutionsClient{
		k8sClient: client,
		namespace: namespace,
	}
}

// TestExecutionsClient client for getting test executions CRs
type TestExecutionsClient struct {
	k8sClient client.Client
	namespace string
}

// Get gets test execution by name in given namespace
func (s TestExecutionsClient) Get(name string) (*testexecutionv1.TestExecution, error) {
	testExecution := &testexecutionv1.TestExecution{}
	if err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: name}, testExecution); err != nil {
		return nil, err
	}

	return testExecution, nil
}

// Create creates new test execution CRD
func (s TestExecutionsClient) Create(testExecution *testexecutionv1.TestExecution) (*testexecutionv1.TestExecution, error) {
	if err := s.k8sClient.Create(context.Background(), testExecution); err != nil {
		return nil, err
	}

	return testExecution, nil
}

// Update updates test execution
func (s TestExecutionsClient) Update(testExecution *testexecutionv1.TestExecution) (*testexecutionv1.TestExecution, error) {
	if err := s.k8sClient.Update(context.Background(), testExecution); err != nil {
		return nil, err
	}

	return testExecution, nil
}

// Delete deletes test execution by name
func (s TestExecutionsClient) Delete(name string) error {
	testExecution := &testexecutionv1.TestExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}

	err := s.k8sClient.Delete(context.Background(), testExecution)
	return err
}

// UpdateStatus updates existing test execution status
func (s TestExecutionsClient) UpdateStatus(testExecution *testexecutionv1.TestExecution) error {
	return s.k8sClient.Status().Update(context.Background(), testExecution)
}
