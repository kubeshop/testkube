package secrets

import (
	"context"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecretClient provide methods to manage secrets
type SecretClient struct {
	ClientSet *kubernetes.Clientset
	Namespace string
	Log       *zap.SugaredLogger
}

// NewSecretClient is a method to create new secret client
func NewSecretClient() (*SecretClient, error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, err
	}

	return &SecretClient{
		ClientSet: clientSet,
		Namespace: "testkube",
		Log:       log.DefaultLogger,
	}, nil
}

// Get is a method to retrieve an existing secret
func (c *SecretClient) Get(id string) (map[string]string, error) {
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
func (c *SecretClient) List() (map[string]map[string]string, error) {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretList, err := secretsClient.List(ctx, metav1.ListOptions{})
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
func (c *SecretClient) Create(id string, stringData map[string]string) error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretSpec := NewSecretSpec(id, c.Namespace, stringData)
	if _, err := secretsClient.Create(ctx, secretSpec, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// Update is a method to update an existing secret
func (c *SecretClient) Update(id string, stringData map[string]string) error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	secretSpec := NewSecretSpec(id, c.Namespace, stringData)
	if _, err := secretsClient.Update(ctx, secretSpec, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// Delete is a method to delete an existing secret
func (c *SecretClient) Delete(id string) error {
	secretsClient := c.ClientSet.CoreV1().Secrets(c.Namespace)
	ctx := context.Background()

	if err := secretsClient.Delete(ctx, id, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// NewSecretSpec is a method to return secret spec
func NewSecretSpec(id, namespace string, stringData map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: namespace,
		},
		Type:       v1.SecretTypeOpaque,
		StringData: stringData,
	}
}
