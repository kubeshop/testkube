package secret

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
)

const testkubeTestSecretLabel = "tests-secrets"

// Client provide methods to manage secrets
type Client struct {
	ClientSet *kubernetes.Clientset
	Log       *zap.SugaredLogger
	Namespace string
}

// NewClient is a method to create new secret client
func NewClient(namespace string) (*Client, error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, err
	}

	return &Client{
		ClientSet: clientSet,
		Log:       log.DefaultLogger,
		Namespace: namespace,
	}, nil
}

// Get is a method to retrieve an existing secret
func (c *Client) Get(id string) (map[string]string, error) {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretSpec, err := secretsClient.Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	stringData := map[string]string{}
	for key, value := range secretSpec.Data {
		stringData[key] = string(value)
	}

	return stringData, nil
}

// List is a method to retrieve all existing secrets
func (c *Client) List() (map[string]map[string]string, error) {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretList, err := secretsClient.List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("testkube=%s", testkubeTestSecretLabel)})
	if err != nil {
		return nil, err
	}

	secretData := map[string]map[string]string{}
	for _, item := range secretList.Items {
		stringData := map[string]string{}
		for key, value := range item.Data {
			stringData[key] = string(value)
		}

		secretData[item.Name] = stringData
	}

	return secretData, nil
}

// Create is a method to create new secret
func (c *Client) Create(id string, stringData map[string]string) error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretSpec := NewSpec(id, c.Namespace, stringData)
	if _, err := secretsClient.Create(ctx, secretSpec, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// Apply is a method to create or update a secret
func (c *Client) Apply(id string, stringData map[string]string) error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretSpec := NewApplySpec(id, c.Namespace, stringData)
	if _, err := secretsClient.Apply(ctx, secretSpec, metav1.ApplyOptions{
		FieldManager: "application/apply-patch"}); err != nil {
		return err
	}

	return nil
}

// Update is a method to update an existing secret
func (c *Client) Update(id string, stringData map[string]string) error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretSpec := NewSpec(id, c.Namespace, stringData)
	if _, err := secretsClient.Update(ctx, secretSpec, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// Delete is a method to delete an existing secret
func (c *Client) Delete(id string) error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	if err := secretsClient.Delete(ctx, id, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// DeleteAll is a method to delete all existing secrets
func (c *Client) DeleteAll() error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	if err := secretsClient.DeleteCollection(ctx, metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: fmt.Sprintf("testkube=%s", testkubeTestSecretLabel)}); err != nil {
		return err
	}

	return nil
}

// NewSpec is a method to return secret spec
func NewSpec(id, namespace string, stringData map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: namespace,
			Labels:    map[string]string{"testkube": testkubeTestSecretLabel},
		},
		Type:       v1.SecretTypeOpaque,
		StringData: stringData,
	}
}

// NewApplySpec is a method to return secret apply spec
func NewApplySpec(id, namespace string, stringData map[string]string) *corev1.SecretApplyConfiguration {
	return corev1.Secret(id, namespace).
		WithLabels(map[string]string{"testkube": testkubeTestSecretLabel}).
		WithStringData(stringData).
		WithType(v1.SecretTypeOpaque)
}

// GetMetadataName returns secret metadata name
func GetMetadataName(name string) string {
	return fmt.Sprintf("%s-secrets", name)
}
