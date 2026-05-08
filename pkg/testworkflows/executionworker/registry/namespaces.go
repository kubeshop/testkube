package registry

import (
	"context"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type namespacesRegistry struct {
	clientSet        kubernetes.Interface
	defaultNamespace string
	namespaces       []string
	cache            *lru.Cache[string, string]
	operations       singleflight.Group
}

type NamespacesRegistry interface {
	Get(ctx context.Context, id string) (string, error)
	Register(id, namespace string)
}

func NewNamespacesRegistry(clientSet kubernetes.Interface, defaultNamespace string, namespaces []string, cacheSize int) NamespacesRegistry {
	cache, _ := lru.New[string, string](cacheSize)
	return &namespacesRegistry{
		clientSet:        clientSet,
		defaultNamespace: defaultNamespace,
		namespaces:       namespaces,
		cache:            cache,
	}
}

func (r *namespacesRegistry) Register(id, namespace string) {
	r.cache.Add(id, namespace)
}

func (r *namespacesRegistry) hasJobAt(ctx context.Context, id, namespace string) (bool, error) {
	// TODO: consider retry
	job, err := r.clientSet.BatchV1().Jobs(namespace).Get(ctx, id, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return job != nil, nil
}

func (r *namespacesRegistry) hasJobTracesAt(ctx context.Context, id, namespace string) (bool, error) {
	// TODO: consider retry
	events, err := r.clientSet.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + id,
		TypeMeta:      metav1.TypeMeta{Kind: "Job"},
		Limit:         1,
	})
	if err != nil {
		return false, err
	}
	return len(events.Items) > 0, nil
}

func (r *namespacesRegistry) load(ctx context.Context, id string) (string, error) {
	// Search firstly for the actual job
	has, err := r.hasJobAt(ctx, id, r.defaultNamespace)
	if err != nil || has {
		return r.defaultNamespace, err
	}
	for _, ns := range r.namespaces {
		has, err = r.hasJobAt(ctx, id, ns)
		if err != nil || has {
			return ns, err
		}
	}

	// Search for the traces
	has, err = r.hasJobTracesAt(ctx, id, r.defaultNamespace)
	if err != nil || has {
		return r.defaultNamespace, err
	}
	for _, ns := range r.namespaces {
		has, err = r.hasJobTracesAt(ctx, id, ns)
		if err != nil || has {
			return ns, err
		}
	}

	// Not found anything
	return "", ErrResourceNotFound
}

func (r *namespacesRegistry) Get(ctx context.Context, id string) (string, error) {
	if ns, ok := r.cache.Get(id); ok {
		return ns, nil
	}

	for {
		obj, err, _ := r.operations.Do(id, func() (interface{}, error) {
			ns, err := r.load(ctx, id)
			if err == nil {
				r.cache.Add(id, ns)
			}
			return ns, err
		})

		// Try again, if context if initial caller has been called
		// TODO: Think how to better use context across multiple callers
		if errors.Is(err, context.Canceled) && ctx.Err() == nil {
			continue
		}

		if err == nil {
			return obj.(string), nil
		}
		return "", err
	}
}
