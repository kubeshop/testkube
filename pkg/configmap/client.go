package configmap

import (
	"context"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
)

//go:generate mockgen -destination=./mock_client.go -package=configmap "github.com/kubeshop/testkube/pkg/configmap" Interface
type Interface interface {
	Get(ctx context.Context, id string, namespace ...string) (map[string]string, error)
	Create(ctx context.Context, id string, stringData map[string]string) error
	Apply(ctx context.Context, id string, stringData map[string]string) error
	Update(ctx context.Context, id string, stringData map[string]string) error
}

// Client provide methods to manage configmaps
type Client struct {
	ClientSet *kubernetes.Clientset
	Log       *zap.SugaredLogger
	Namespace string
}

// NewClient is a method to create new configmap client
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

// Create is a method to create new configmap
func (c *Client) Create(ctx context.Context, id string, stringData map[string]string) error {
	configMapsClient := c.ClientSet.CoreV1().ConfigMaps(c.Namespace)

	configMapSpec := NewSpec(id, c.Namespace, stringData)
	if _, err := configMapsClient.Create(ctx, configMapSpec, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// Get is a method to retrieve an existing configmap
func (c *Client) Get(ctx context.Context, id string, namespace ...string) (map[string]string, error) {
	ns := c.Namespace
	if len(namespace) != 0 {
		ns = namespace[0]
	}

	configMapsClient := c.ClientSet.CoreV1().ConfigMaps(ns)

	configMapSpec, err := configMapsClient.Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	stringData := map[string]string{}
	for key, value := range configMapSpec.Data {
		stringData[key] = value
	}

	return stringData, nil
}

// Update is a method to update an existing configmap
func (c *Client) Update(ctx context.Context, id string, stringData map[string]string) error {
	configMapsClient := c.ClientSet.CoreV1().ConfigMaps(c.Namespace)

	configMapSpec := NewSpec(id, c.Namespace, stringData)
	if _, err := configMapsClient.Update(ctx, configMapSpec, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// Apply is a method to create or update a configmap
func (c *Client) Apply(ctx context.Context, id string, stringData map[string]string) error {
	configMapsClient := c.ClientSet.CoreV1().ConfigMaps(c.Namespace)

	configMapSpec := NewApplySpec(id, c.Namespace, stringData)
	if _, err := configMapsClient.Apply(ctx, configMapSpec, metav1.ApplyOptions{
		FieldManager: "application/apply-patch"}); err != nil {
		return err
	}

	return nil
}

// NewSpec is a method to return configmap spec
func NewSpec(id, namespace string, stringData map[string]string) *v1.ConfigMap {
	configuration := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: namespace,
		},
		Data: stringData,
	}

	return configuration
}

// NewApplySpec is a method to return configmap apply spec
func NewApplySpec(id, namespace string, stringData map[string]string) *corev1.ConfigMapApplyConfiguration {
	configuration := corev1.ConfigMap(id, namespace).
		WithData(stringData)

	return configuration
}
