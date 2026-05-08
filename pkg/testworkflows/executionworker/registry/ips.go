package registry

import (
	"context"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

type ipsRegistry struct {
	clientSet    kubernetes.Interface
	getNamespace func(ctx context.Context, id string) (string, error)
	cache        *lru.Cache[string, string]
	operations   singleflight.Group
}

type PodIpsRegistry interface {
	Get(ctx context.Context, id string) (string, error)
	Register(id, podIp string)
}

func NewPodIpsRegistry(clientSet kubernetes.Interface, getNamespace func(ctx context.Context, id string) (string, error), cacheSize int) PodIpsRegistry {
	cache, _ := lru.New[string, string](cacheSize)
	return &ipsRegistry{
		clientSet:    clientSet,
		getNamespace: getNamespace,
		cache:        cache,
	}
}

func (r *ipsRegistry) Register(id, podIp string) {
	r.cache.Add(id, podIp)
}

func (r *ipsRegistry) load(ctx context.Context, id string) (string, error) {
	ns, err := r.getNamespace(ctx, id)
	if err != nil {
		return "", err
	}
	// TODO: consider retry
	pods, err := r.clientSet.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: constants.ResourceIdLabelName + "=" + id,
		Limit:         1,
	})
	if err != nil {
		return "", err
	}
	return pods.Items[0].Status.PodIP, nil
}

func (r *ipsRegistry) Get(ctx context.Context, id string) (string, error) {
	if ns, ok := r.cache.Get(id); ok {
		return ns, nil
	}

	for {
		obj, err, _ := r.operations.Do(id, func() (interface{}, error) {
			ip, err := r.load(ctx, id)
			if ip != "" && err == nil {
				r.cache.Add(id, ip)
			}
			return ip, err
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
