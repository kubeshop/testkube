package imageinspector

import (
	"context"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/cache"
	"github.com/kubeshop/testkube/pkg/log"

	"github.com/kubeshop/testkube/pkg/secret"
)

type SecretFetcherOption func(*secretFetcher)

// WithSecretCacheTTL sets the time to live for the cached secrets.
func WithSecretCacheTTL(ttl time.Duration) SecretFetcherOption {
	return func(s *secretFetcher) {
		s.ttl = ttl
	}
}

type secretFetcher struct {
	client secret.Interface
	cache  cache.Cache[*corev1.Secret]
	ttl    time.Duration
}

func NewSecretFetcher(client secret.Interface, cache cache.Cache[*corev1.Secret], opts ...SecretFetcherOption) SecretFetcher {
	s := &secretFetcher{
		client: client,
		cache:  cache,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *secretFetcher) Get(ctx context.Context, name string) (*corev1.Secret, error) {
	if s.ttl > 0 {
		// Get cached secret
		cached, err := s.getFromCache(ctx, name)
		if err != nil {
			return nil, err
		}
		if cached != nil {
			return cached, nil
		}
	}

	// Load secret from the Kubernetes
	obj, err := s.client.GetObject(name)
	if err != nil {
		return nil, errors.Wrap(err, "fetching image pull secret")
	}

	if s.ttl > 0 {
		// Save in cache
		if err := s.cache.Set(ctx, name, obj, s.ttl); err != nil {
			log.DefaultLogger.Warnw("error while saving secret in cache", "name", name, "error", err)
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return obj, nil
}

func (s *secretFetcher) getFromCache(ctx context.Context, name string) (*corev1.Secret, error) {
	cached, err := s.cache.Get(ctx, name)
	if err != nil {
		if cache.IsCacheMiss(err) {
			return nil, nil
		}
		log.DefaultLogger.Warnw("error while getting secret from cache", "name", name, "error", err)
		return nil, err
	}

	return cached, nil
}
