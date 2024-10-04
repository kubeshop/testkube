package testworkflowcontroller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	DefaultTimeoutSeconds = int64(365 * 24 * 3600)
)

type kubernetesClient[T any, U any] interface {
	List(ctx context.Context, options metav1.ListOptions) (*T, error)
	Watch(ctx context.Context, options metav1.ListOptions) (watch.Interface, error)
}
