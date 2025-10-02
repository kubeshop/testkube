package secret

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// TestkubeTestSecretLabel is testkube test secrets label
	TestkubeTestSecretLabel = "tests-secrets"
	// TestkubeTestSourcesSecretLabel is testkube test sources secrets label
	TestkubeTestSourcesSecretLabel = "test-sources-secrets"
)

// Client provide methods to manage secrets
type Client struct {
	client.Client
	namespace string
	label     string
}

// NewClient is a method to create new secret client
func NewClient(cli client.Client, namespace, label string) *Client {
	return &Client{
		Client:    cli,
		namespace: namespace,
		label:     label,
	}
}

// Get is a method to retrieve an existing secret
func (c *Client) Get(id string) (map[string]string, error) {
	secret := &corev1.Secret{}
	ctx := context.Background()

	if err := c.Client.Get(ctx, client.ObjectKey{
		Namespace: c.namespace, Name: id}, secret); err != nil {
		return nil, err
	}

	stringData := map[string]string{}
	for key, value := range secret.Data {
		stringData[key] = string(value)
	}

	return stringData, nil
}

// Create is a method to create new secret
func (c *Client) Create(id string, labels, stringData map[string]string) error {
	ctx := context.Background()
	secretSpec := NewSpec(id, c.namespace, c.label, labels, stringData)
	if err := c.Client.Create(ctx, secretSpec); err != nil {
		return err
	}

	return nil
}

// Update is a method to update an existing secret
func (c *Client) Update(id string, labels, stringData map[string]string) error {
	ctx := context.Background()
	secretSpec := NewSpec(id, c.namespace, c.label, labels, stringData)
	if err := c.Client.Update(ctx, secretSpec); err != nil {
		return err
	}

	return nil
}

// Apply is a method to create new secret or update an existing secret
func (c *Client) Apply(id string, labels, stringData map[string]string) error {
	secret := &corev1.Secret{}
	ctx := context.Background()

	err := c.Client.Get(ctx, client.ObjectKey{Namespace: c.namespace, Name: id}, secret)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		return c.Create(id, labels, stringData)
	}

	return c.Update(id, labels, stringData)
}

// Delete is a method to delete an existing secret
func (c *Client) Delete(id string) error {
	ctx := context.Background()
	secret := &corev1.Secret{}

	if err := c.Client.Get(ctx, client.ObjectKey{
		Namespace: c.namespace, Name: id}, secret); err != nil {
		return err
	}

	if err := c.Client.Delete(ctx, secret); err != nil {
		return err
	}

	return nil
}

// DeleteAll is a method to delete all existing secrets
func (c *Client) DeleteAll(selector string) error {
	ctx := context.Background()
	filter := fmt.Sprintf("testkube=%s", c.label)
	if selector != "" {
		filter += "," + selector
	}

	reqs, err := labels.ParseToRequirements(filter)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("Secret")
	u.SetAPIVersion("v1")
	err = c.Client.DeleteAllOf(ctx, u, client.InNamespace(c.namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})

	return err
}

// NewSpec is a method to return secret spec
func NewSpec(id, namespace, label string, labels, stringData map[string]string) *v1.Secret {
	configuration := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: namespace,
			Labels:    map[string]string{"testkube": label, "createdBy": "testkube"},
		},
		Type:       v1.SecretTypeOpaque,
		StringData: stringData,
	}

	for key, value := range labels {
		configuration.Labels[key] = value
	}

	return configuration
}

// GetMetadataName returns secret metadata name
func GetMetadataName(name, kind string) string {
	return fmt.Sprintf("%s-%s", name, kind)
}
