package configmap

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -destination=./mock_client.go -package=configmap "github.com/kubeshop/testkube/pkg/operator/configmap" Interface
type Interface interface {
	Get(ctx context.Context, name, namespace string) (map[string]string, error)
	ListAll(ctx context.Context, selector, namespace string) (*corev1.ConfigMapList, error)
}

// Client provide methods to manage configmaps
type Client struct {
	client.Client
}

// New is a method to create new configmap client
func New(cli client.Client) *Client {
	return &Client{
		Client: cli,
	}
}

// Get is a method to retrieve an existing configmap
func (c *Client) Get(ctx context.Context, name, namespace string) (map[string]string, error) {
	var configMap corev1.ConfigMap
	if err := c.Client.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace}, &configMap); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	stringData := make(map[string]string)
	for key, value := range configMap.Data {
		stringData[key] = value
	}

	return stringData, nil
}

// ListAll is a method to list all configmaps by selector
func (c *Client) ListAll(ctx context.Context, selector, namespace string) (*corev1.ConfigMapList, error) {
	list := &corev1.ConfigMapList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}
	if err = c.List(context.Background(), list, options); err != nil {
		return list, err
	}

	return list, nil
}
