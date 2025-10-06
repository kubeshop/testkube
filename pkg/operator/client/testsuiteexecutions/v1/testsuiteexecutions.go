package testsuiteexecutions

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testsuiteexecutionv1 "github.com/kubeshop/testkube/api/testsuiteexecution/v1"
)

//go:generate mockgen -destination=./mock_testsuiteexecutions.go -package=testsuiteexecutions "github.com/kubeshop/testkube/pkg/operator/client/testsuiteexecutions/v1" Interface
type Interface interface {
	Get(name string) (*testsuiteexecutionv1.TestSuiteExecution, error)
	Create(testSuiteExecution *testsuiteexecutionv1.TestSuiteExecution) (*testsuiteexecutionv1.TestSuiteExecution, error)
	Update(testSuiteExecution *testsuiteexecutionv1.TestSuiteExecution) (*testsuiteexecutionv1.TestSuiteExecution, error)
	Delete(name string) error
	UpdateStatus(testSuiteЕxecution *testsuiteexecutionv1.TestSuiteExecution) error
}

// NewClient returns new client instance, needs kubernetes client to be passed as dependecy
func NewClient(client client.Client, namespace string) *TestSuiteExecutionsClient {
	return &TestSuiteExecutionsClient{
		k8sClient: client,
		namespace: namespace,
	}
}

// TestSuiteExecutionsClient client for getting test suite executions CRs
type TestSuiteExecutionsClient struct {
	k8sClient client.Client
	namespace string
}

// Get gets test suite execution by name in given namespace
func (s TestSuiteExecutionsClient) Get(name string) (*testsuiteexecutionv1.TestSuiteExecution, error) {
	testSuiteExecution := &testsuiteexecutionv1.TestSuiteExecution{}
	if err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: name}, testSuiteExecution); err != nil {
		return nil, err
	}

	return testSuiteExecution, nil
}

// Create creates new test suite execution CRD
func (s TestSuiteExecutionsClient) Create(testSuiteExecution *testsuiteexecutionv1.TestSuiteExecution) (*testsuiteexecutionv1.TestSuiteExecution, error) {
	if err := s.k8sClient.Create(context.Background(), testSuiteExecution); err != nil {
		return nil, err
	}

	return testSuiteExecution, nil
}

// Update updates test suite execution
func (s TestSuiteExecutionsClient) Update(testSuiteExecution *testsuiteexecutionv1.TestSuiteExecution) (*testsuiteexecutionv1.TestSuiteExecution, error) {
	if err := s.k8sClient.Update(context.Background(), testSuiteExecution); err != nil {
		return nil, err
	}

	return testSuiteExecution, nil
}

// Delete deletes test suite execution by name
func (s TestSuiteExecutionsClient) Delete(name string) error {
	testSuiteExecution := &testsuiteexecutionv1.TestSuiteExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}

	err := s.k8sClient.Delete(context.Background(), testSuiteExecution)
	return err
}

// UpdateStatus updates existing test suite execution status
func (s TestSuiteExecutionsClient) UpdateStatus(testSuiteExecution *testsuiteexecutionv1.TestSuiteExecution) error {
	return s.k8sClient.Status().Update(context.Background(), testSuiteExecution)
}
