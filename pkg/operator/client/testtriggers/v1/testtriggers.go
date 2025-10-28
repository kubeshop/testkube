package v1

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

//go:generate go tool mockgen -destination=./mock_testtriggers.go -package=v1 "github.com/kubeshop/testkube/pkg/operator/client/testtriggers/v1" Interface
type Interface interface {
	List(selector, namespace string) (*testtriggersv1.TestTriggerList, error)
	Get(name, namespace string) (*testtriggersv1.TestTrigger, error)
	Create(trigger *testtriggersv1.TestTrigger) (*testtriggersv1.TestTrigger, error)
	Update(trigger *testtriggersv1.TestTrigger) (*testtriggersv1.TestTrigger, error)
	Delete(name, namespace string) error
	DeleteAll(namespace string) error
	DeleteByLabels(selector, namespace string) error
}

// NewClient creates new TestTrigger client
func NewClient(client client.Client, namespace string) *TestTriggersClient {
	return &TestTriggersClient{
		Client:    client,
		Namespace: namespace,
	}
}

// TestTriggersClient implements methods to work with TestTriggers
type TestTriggersClient struct {
	Client    client.Client
	Namespace string
}

// List lists TestTriggers
func (s TestTriggersClient) List(selector, namespace string) (*testtriggersv1.TestTriggerList, error) {
	list := &testtriggersv1.TestTriggerList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}
	if namespace == "" {
		namespace = s.Namespace
	}

	options := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}

	if err = s.Client.List(context.Background(), list, options); err != nil {
		return list, err
	}

	return list, nil
}

// Get returns TestTrigger
func (s TestTriggersClient) Get(name, namespace string) (*testtriggersv1.TestTrigger, error) {
	if namespace == "" {
		namespace = s.Namespace
	}
	trigger := &testtriggersv1.TestTrigger{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, trigger)
	if err != nil {
		return nil, err
	}
	return trigger, nil
}

// Create creates new TestTrigger
func (s TestTriggersClient) Create(trigger *testtriggersv1.TestTrigger) (*testtriggersv1.TestTrigger, error) {
	return trigger, s.Client.Create(context.Background(), trigger)
}

// Update updates existing TestTrigger
func (s TestTriggersClient) Update(trigger *testtriggersv1.TestTrigger) (*testtriggersv1.TestTrigger, error) {
	return trigger, s.Client.Update(context.Background(), trigger)
}

// Delete deletes existing TestTrigger
func (s TestTriggersClient) Delete(name, namespace string) error {
	trigger, err := s.Get(name, namespace)
	if err != nil {
		return err
	}
	return s.Client.Delete(context.Background(), trigger)
}

// DeleteAll delete all TestTriggers
func (s TestTriggersClient) DeleteAll(namespace string) error {
	if namespace == "" {
		namespace = s.Namespace
	}
	u := &unstructured.Unstructured{}
	u.SetKind("TestTrigger")
	u.SetAPIVersion(testtriggersv1.GroupVersion.String())
	return s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(namespace))
}

// DeleteByLabels deletes TestTriggers by labels
func (s TestTriggersClient) DeleteByLabels(selector, namespace string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}
	if namespace == "" {
		namespace = s.Namespace
	}

	u := &unstructured.Unstructured{}
	u.SetKind("TestTrigger")
	u.SetAPIVersion(testtriggersv1.GroupVersion.String())
	err = s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}
