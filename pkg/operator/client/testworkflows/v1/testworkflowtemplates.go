package v1

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

//go:generate mockgen -destination=./mock_testworkflowtemplates.go -package=v1 "github.com/kubeshop/testkube/pkg/operator/client/testworkflows/v1" TestWorkflowTemplatesInterface
type TestWorkflowTemplatesInterface interface {
	List(selector string) (*testworkflowsv1.TestWorkflowTemplateList, error)
	ListLabels() (map[string][]string, error)
	Get(name string) (*testworkflowsv1.TestWorkflowTemplate, error)
	Create(template *testworkflowsv1.TestWorkflowTemplate) (*testworkflowsv1.TestWorkflowTemplate, error)
	Update(template *testworkflowsv1.TestWorkflowTemplate) (*testworkflowsv1.TestWorkflowTemplate, error)
	Apply(template *testworkflowsv1.TestWorkflowTemplate) error
	Delete(name string) error
	DeleteAll() error
	DeleteByLabels(selector string) error
	UpdateStatus(template *testworkflowsv1.TestWorkflowTemplate) error
}

// NewTestWorkflowTemplatesClient creates new TestWorkflowTemplate client
func NewTestWorkflowTemplatesClient(client client.Client, namespace string) *TestWorkflowTemplatesClient {
	return &TestWorkflowTemplatesClient{
		Client:    client,
		Namespace: namespace,
	}
}

// TestWorkflowTemplatesClient implements methods to work with TestWorkflowTemplates
type TestWorkflowTemplatesClient struct {
	Client    client.Client
	Namespace string
}

// List lists TestWorkflowTemplates
func (s TestWorkflowTemplatesClient) List(selector string) (*testworkflowsv1.TestWorkflowTemplateList, error) {
	list := &testworkflowsv1.TestWorkflowTemplateList{}
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

// ListLabels lists labels for TestWorkflowTemplates
func (s TestWorkflowTemplatesClient) ListLabels() (map[string][]string, error) {
	labels := map[string][]string{}
	list := &testworkflowsv1.TestWorkflowTemplateList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	if err != nil {
		return labels, err
	}

	for _, template := range list.Items {
		for key, value := range template.Labels {
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

// Get returns TestWorkflowTemplate
func (s TestWorkflowTemplatesClient) Get(name string) (*testworkflowsv1.TestWorkflowTemplate, error) {
	template := &testworkflowsv1.TestWorkflowTemplate{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, template)
	if err != nil {
		return nil, err
	}
	return template, nil
}

// Create creates new TestWorkflowTemplate
func (s TestWorkflowTemplatesClient) Create(template *testworkflowsv1.TestWorkflowTemplate) (*testworkflowsv1.TestWorkflowTemplate, error) {
	return template, s.Client.Create(context.Background(), template)
}

// Update updates existing TestWorkflowTemplate
func (s TestWorkflowTemplatesClient) Update(template *testworkflowsv1.TestWorkflowTemplate) (*testworkflowsv1.TestWorkflowTemplate, error) {
	return template, s.Client.Update(context.Background(), template)
}

// Apply applies changes to the existing TestWorkflowTemplate
func (s TestWorkflowTemplatesClient) Apply(template *testworkflowsv1.TestWorkflowTemplate) error {
	return s.Client.Patch(context.Background(), template, client.Apply, &client.PatchOptions{
		FieldManager: "application/apply-patch",
	})
}

// Delete deletes existing TestWorkflowTemplate
func (s TestWorkflowTemplatesClient) Delete(name string) error {
	template, err := s.Get(name)
	if err != nil {
		return err
	}
	return s.Client.Delete(context.Background(), template)
}

// DeleteAll delete all TestWorkflowTemplates
func (s TestWorkflowTemplatesClient) DeleteAll() error {
	u := &unstructured.Unstructured{}
	u.SetKind("TestWorkflowTemplate")
	u.SetAPIVersion(testworkflowsv1.GroupVersion.String())
	return s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace))
}

// UpdateStatus updates existing TestWorkflowTemplate status
func (s TestWorkflowTemplatesClient) UpdateStatus(template *testworkflowsv1.TestWorkflowTemplate) error {
	return s.Client.Status().Update(context.Background(), template)
}

// DeleteByLabels deletes TestWorkflowTemplates by labels
func (s TestWorkflowTemplatesClient) DeleteByLabels(selector string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("TestWorkflowTemplate")
	u.SetAPIVersion(testworkflowsv1.GroupVersion.String())
	err = s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}
