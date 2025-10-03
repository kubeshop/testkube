package imageinspector

import (
	"context"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/cache"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/pkg/secret"
)

func TestSecretFetcherGetExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := secret.NewMockInterface(ctrl)
	fetcher := NewSecretFetcher(client, cache.NewInMemoryCache[*corev1.Secret]())

	expected := corev1.Secret{
		StringData: map[string]string{"key": "value"},
	}
	client.EXPECT().GetObject("dummy").Return(&expected, nil)

	result, err := fetcher.Get(context.Background(), "dummy")
	assert.NoError(t, err)
	assert.Equal(t, &expected, result)
}

func TestSecretFetcherGetCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := secret.NewMockInterface(ctrl)
	fetcher := NewSecretFetcher(client, cache.NewInMemoryCache[*corev1.Secret](), WithSecretCacheTTL(1*time.Minute))

	expected := corev1.Secret{
		StringData: map[string]string{"key": "value"},
	}
	client.EXPECT().GetObject("dummy").Return(&expected, nil)

	result1, err1 := fetcher.Get(context.Background(), "dummy")
	result2, err2 := fetcher.Get(context.Background(), "dummy")
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, &expected, result1)
	assert.Equal(t, &expected, result2)
}

func TestSecretFetcherGetDisabledCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := secret.NewMockInterface(ctrl)
	fetcher := NewSecretFetcher(client, newNoCache(t), WithSecretCacheTTL(0))

	expected := corev1.Secret{
		StringData: map[string]string{"key": "value"},
	}
	client.EXPECT().GetObject("dummy").Return(&expected, nil)

	result1, err1 := fetcher.Get(context.Background(), "dummy")
	assert.NoError(t, err1)
	assert.Equal(t, &expected, result1)
}

func TestSecretFetcherGetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := secret.NewMockInterface(ctrl)
	fetcher := NewSecretFetcher(client, cache.NewInMemoryCache[*corev1.Secret]())

	client.EXPECT().GetObject("dummy").Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, "dummy"))
	client.EXPECT().GetObject("dummy").Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, "dummy"))

	result1, err1 := fetcher.Get(context.Background(), "dummy")
	result2, err2 := fetcher.Get(context.Background(), "dummy")
	var noSecret *corev1.Secret
	assert.Error(t, err1)
	assert.Error(t, err2)
	assert.True(t, k8serrors.IsNotFound(err1))
	assert.True(t, k8serrors.IsNotFound(err2))
	assert.Equal(t, noSecret, result1)
	assert.Equal(t, noSecret, result2)
}

type noCache struct {
	t *testing.T
}

func newNoCache(t *testing.T) *noCache {
	return &noCache{t: t}
}

func (n *noCache) Set(ctx context.Context, key string, value *corev1.Secret, ttl time.Duration) error {
	n.t.Fatalf("set method should not be invoked when cache is disabled")
	return nil
}

func (n *noCache) Get(ctx context.Context, key string) (*corev1.Secret, error) {
	n.t.Fatalf("get method should not be invoked when cache is disabled")
	return nil, nil
}
