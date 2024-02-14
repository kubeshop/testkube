package imageinspector

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/secret"
)

type secretFetcher struct {
	client secret.Interface
	cache  map[string]*corev1.Secret
	mu     sync.RWMutex
}

func NewSecretFetcher(client secret.Interface) SecretFetcher {
	return &secretFetcher{
		client: client,
		cache:  make(map[string]*corev1.Secret),
	}
}

func (s *secretFetcher) Get(ctx context.Context, name string) (*corev1.Secret, error) {
	// Get cached secret
	s.mu.RLock()
	if v, ok := s.cache[name]; ok {
		s.mu.RUnlock()
		return v, nil
	}
	s.mu.RUnlock()

	// Load secret from the Kubernetes
	obj, err := s.client.GetObject(name)
	if err != nil {
		return nil, errors.Wrap(err, "fetching image pull secret")
	}

	// Save in cache
	s.mu.Lock()
	s.cache[name] = obj
	s.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return obj, nil
}
