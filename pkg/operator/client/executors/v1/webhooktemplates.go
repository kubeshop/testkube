package executors

import (
	"context"
	"fmt"

	executorsv1 "github.com/kubeshop/testkube/api/executor/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate go tool mockgen -destination=./mock_webhooktemplates.go -package=executors "github.com/kubeshop/testkube/pkg/operator/client/executors/v1" WebhookTemplatesInterface
type WebhookTemplatesInterface interface {
	List(selector string) (*executorsv1.WebhookTemplateList, error)
	Get(name string) (*executorsv1.WebhookTemplate, error)
	GetByEvent(event executorsv1.EventType) (*executorsv1.WebhookTemplateList, error)
	Create(webhookTemplate *executorsv1.WebhookTemplate) (*executorsv1.WebhookTemplate, error)
	Update(webhookTemplate *executorsv1.WebhookTemplate) (*executorsv1.WebhookTemplate, error)
	Delete(name string) error
	DeleteByLabels(selector string) error
}

// NewWebhookTemplatesClient returns new client instance, needs kubernetes client to be passed as dependecy
func NewWebhookTemplatesClient(client client.Client, namespace string) *WebhookTemplatesClient {
	return &WebhookTemplatesClient{
		Client:    client,
		Namespace: namespace,
	}
}

// WebhookTemplatesClient client for getting webhook templates CRs
type WebhookTemplatesClient struct {
	Client    client.Client
	Namespace string
}

// List shows list of available webhook templates
func (s WebhookTemplatesClient) List(selector string) (*executorsv1.WebhookTemplateList, error) {
	list := &executorsv1.WebhookTemplateList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		Namespace:     s.Namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}

	err = s.Client.List(context.Background(), list, options)
	return list, err
}

// Get gets webhook template by name in given namespace
func (s WebhookTemplatesClient) Get(name string) (*executorsv1.WebhookTemplate, error) {
	item := &executorsv1.WebhookTemplate{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, item)
	return item, err
}

// GetByEvent gets all webhook templates with given event
func (s WebhookTemplatesClient) GetByEvent(event executorsv1.EventType) (*executorsv1.WebhookTemplateList, error) {
	list := &executorsv1.WebhookTemplateList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	if err != nil {
		return nil, err
	}

	for i := len(list.Items) - 1; i >= 0; i-- {
		exec := list.Items[i]
		hasEvent := false
		for _, t := range exec.Spec.Events {
			if t == event {
				hasEvent = true
			}
		}

		if !hasEvent {
			list.Items = append(list.Items[:i], list.Items[i+1:]...)
		}
	}

	return list, nil
}

// Create creates new Webhook Template CR
func (s WebhookTemplatesClient) Create(webhookTemplate *executorsv1.WebhookTemplate) (*executorsv1.WebhookTemplate, error) {
	if webhookTemplate.Namespace != s.Namespace {
		return nil, fmt.Errorf("wrong namespace, expected: %s, got: %s", s.Namespace, webhookTemplate.Namespace)
	}
	err := s.Client.Create(context.Background(), webhookTemplate)
	if err != nil {
		return nil, fmt.Errorf("could not create webhook template: %w", err)
	}
	res, err := s.Get(webhookTemplate.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get created webhook template: %w", err)
	}
	return res, nil
}

// Delete deletes Webhook Template by name
func (s WebhookTemplatesClient) Delete(name string) error {
	webhookTemplate := &executorsv1.WebhookTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.Namespace,
		},
	}
	err := s.Client.Delete(context.Background(), webhookTemplate)
	return err
}

// Update updates Webhook Template
func (s WebhookTemplatesClient) Update(webhookTemplate *executorsv1.WebhookTemplate) (*executorsv1.WebhookTemplate, error) {
	err := s.Client.Update(context.Background(), webhookTemplate)
	return webhookTemplate, err
}

// DeleteByLabels deletes webhook templates by labels
func (s WebhookTemplatesClient) DeleteByLabels(selector string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("WebhookTemplate")
	u.SetAPIVersion("executor.testkube.io/v1")
	err = s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}
