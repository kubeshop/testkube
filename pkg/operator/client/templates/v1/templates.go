package templates

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	templatev1 "github.com/kubeshop/testkube/api/template/v1"
)

//go:generate go tool mockgen -destination=./mock_templates.go -package=templates "github.com/kubeshop/testkube/pkg/operator/client/templates/v1" Interface
type Interface interface {
	List(selector string) (*templatev1.TemplateList, error)
	Get(name string) (*templatev1.Template, error)
	Create(template *templatev1.Template) (*templatev1.Template, error)
	Update(template *templatev1.Template) (*templatev1.Template, error)
	Delete(name string) error
	DeleteByLabels(selector string) error
}

// NewClient returns new client instance, needs kubernetes client to be passed as dependecy
func NewClient(client client.Client, namespace string) *TemplatesClient {
	return &TemplatesClient{
		k8sClient: client,
		namespace: namespace,
	}
}

// TemplatesClient client for getting templates CRs
type TemplatesClient struct {
	k8sClient client.Client
	namespace string
}

// List shows list of available executors
func (s TemplatesClient) List(selector string) (*templatev1.TemplateList, error) {
	list := &templatev1.TemplateList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		Namespace:     s.namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}

	err = s.k8sClient.List(context.Background(), list, options)
	return list, err
}

// Get gets template by name in given namespace
func (s TemplatesClient) Get(name string) (*templatev1.Template, error) {
	template := &templatev1.Template{}
	if err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: name}, template); err != nil {
		return nil, err
	}

	return template, nil
}

// Create creates new template CRD
func (s TemplatesClient) Create(template *templatev1.Template) (*templatev1.Template, error) {
	if err := s.k8sClient.Create(context.Background(), template); err != nil {
		return nil, err
	}

	return template, nil
}

// Update updates template
func (s TemplatesClient) Update(template *templatev1.Template) (*templatev1.Template, error) {
	if err := s.k8sClient.Update(context.Background(), template); err != nil {
		return nil, err
	}

	return template, nil
}

// Delete deletes template by name
func (s TemplatesClient) Delete(name string) error {
	template := &templatev1.Template{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}

	err := s.k8sClient.Delete(context.Background(), template)
	return err
}

// DeleteByLabels deletes templates by labels
func (s TemplatesClient) DeleteByLabels(selector string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("Template")
	u.SetAPIVersion("tests.testkube.io/v1")
	err = s.k8sClient.DeleteAllOf(context.Background(), u, client.InNamespace(s.namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}
