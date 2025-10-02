// TODO DEPRECATED migrated to executors/v1 package
package executors

import (
	"context"
	"fmt"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClient returns new client instance, needs kubernetes client to be passed as dependecy
func NewClient(client client.Client, namespace string) *ExecutorsClient {
	return &ExecutorsClient{
		Client:    client,
		Namespace: namespace,
	}
}

// ExecutorsClient client for getting executors CRs
type ExecutorsClient struct {
	Client    client.Client
	Namespace string
}

// List shows list of available executors
func (s ExecutorsClient) List() (*executorv1.ExecutorList, error) {
	list := &executorv1.ExecutorList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	return list, err
}

// Get gets executor by name in given namespace
func (s ExecutorsClient) Get(name string) (*executorv1.Executor, error) {
	executor := &executorv1.Executor{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, executor)
	return executor, err
}

// GetByType gets first available executor for given type
func (s ExecutorsClient) GetByType(executorType string) (*executorv1.Executor, error) {
	list := &executorv1.ExecutorList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, exec := range list.Items {
		names = append(names, fmt.Sprintf("%s/%s", exec.Namespace, exec.Name))
		for _, t := range exec.Spec.Types {
			if t == executorType {
				return &exec, nil
			}
		}
	}

	return nil, fmt.Errorf("executor type '%s' is not handled by any of executors (%s)", executorType, names)
}

// Create creates new Executor CR
func (s ExecutorsClient) Create(executor *executorv1.Executor) (*executorv1.Executor, error) {
	err := s.Client.Create(context.Background(), executor)
	return executor, err
}

// Delete deletes executor by name
func (s ExecutorsClient) Delete(name string) error {
	executor := &executorv1.Executor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.Namespace,
		},
	}
	err := s.Client.Delete(context.Background(), executor)
	return err
}

// Update updates executor
func (s ExecutorsClient) Update(executor *executorv1.Executor) (*executorv1.Executor, error) {
	err := s.Client.Update(context.Background(), executor)
	return executor, err
}
