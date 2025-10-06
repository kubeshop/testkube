package namespace

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -destination=./mock_client.go -package=namespace "github.com/kubeshop/testkube/pkg/operator/namespace" Interface
type Interface interface {
	ListAll(ctx context.Context, selector string) (*corev1.NamespaceList, error)
}

// Client provide methods to manage namespaces
type Client struct {
	client.Client
}

// New is a method to create new namespace client
func New(cli client.Client) *Client {
	return &Client{
		Client: cli,
	}
}

// ListAll is a method to list all namespaces by selector
func (c *Client) ListAll(ctx context.Context, selector string) (*corev1.NamespaceList, error) {
	list := &corev1.NamespaceList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(reqs...),
	}
	if err = c.List(context.Background(), list, options); err != nil {
		return list, err
	}

	return list, nil
}
