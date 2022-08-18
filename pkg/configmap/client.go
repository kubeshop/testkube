package configmap

import (
	"context"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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

// Get is a method to retrieve an existing configmap
func (c *Client) Get(id string) (map[string]string, error) {
	configMapsClient := c.ClientSet.CoreV1().ConfigMaps(c.Namespace)
	ctx := context.Background()

	configMapSpec, err := configMapsClient.Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	stringData := map[string]string{}
	for key, value := range configMapSpec.Data {
		stringData[key] = string(value)
	}

	return stringData, nil
}

// Update is a method to update an existing sconfigmap
func (c *Client) Update(id string, stringData map[string]string) error {
	configMapsClient := c.ClientSet.CoreV1().ConfigMaps(c.Namespace)
	ctx := context.Background()

	configMapSpec := NewSpec(id, c.Namespace, stringData)
	if _, err := configMapsClient.Update(ctx, configMapSpec, metav1.UpdateOptions{}); err != nil {
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
